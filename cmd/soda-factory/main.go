// soda-factory is the private operator interface to supervised factory runs:
// status reads recorded state, stop retires one run, and reconcile settles
// outstanding runs. It addresses the dashboard backend's operator endpoint
// over a restricted Unix socket and keeps no database of its own.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "soda-factory:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("soda-factory", flag.ContinueOnError)
	socket := flags.String("socket", "", "private backend operator Unix socket")
	command := flags.String("command", "", "client-generated command identity for stop and reconcile")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *socket == "" {
		return errors.New("usage: soda-factory --socket PATH [--command ID] status [RUN] / stop RUN / reconcile")
	}
	client := &http.Client{Timeout: 11 * time.Minute, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", *socket)
	}}}
	return dispatch(ctx, client, *command, flags.Args(), out)
}

func dispatch(ctx context.Context, client *http.Client, command string, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: soda-factory --socket PATH [--command ID] status [RUN] / stop RUN / reconcile")
	}
	var envelope map[string]string
	switch args[0] {
	case "status":
		if len(args) > 2 || (len(args) == 2 && args[1] == "") {
			return errors.New("status takes an optional run identity")
		}
		if command != "" {
			return errors.New("status carries no durable command")
		}
		envelope = map[string]string{"type": "status"}
		if len(args) == 2 {
			envelope["target"] = args[1]
		}
	case "stop":
		if len(args) != 2 || command == "" {
			return errors.New("stop requires --command ID and one recorded run")
		}
		envelope = map[string]string{"type": "stop", "command_id": command, "target": args[1]}
	case "reconcile":
		if len(args) != 1 || command == "" {
			return errors.New("reconcile requires --command ID and no target")
		}
		envelope = map[string]string{"type": "reconcile", "command_id": command}
	default:
		return errors.New("unknown factory command")
	}
	return send(ctx, client, envelope, out)
}

func send(ctx context.Context, client *http.Client, envelope map[string]string, out io.Writer) error {
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(envelope); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://soda-operator/operator/factory", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return errors.New("operator endpoint unavailable")
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(res.Body, 256<<10+1))
	if err != nil || len(data) > 256<<10 {
		return errors.New("operator response exceeds limit")
	}
	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		if _, err = out.Write(data); err != nil {
			return err
		}
		if len(data) == 0 || data[len(data)-1] != '\n' {
			_, _ = io.WriteString(out, "\n")
		}
		return nil
	case http.StatusNotFound:
		return errors.New("factory run not found")
	case http.StatusConflict:
		return errors.New("command identity reused for different content")
	default:
		return fmt.Errorf("operator command failed (HTTP %d)", res.StatusCode)
	}
}
