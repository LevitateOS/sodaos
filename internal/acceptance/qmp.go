// Adapted from soda-os bc1d3e0 internal/acceptance/qmp.go.
package acceptance

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type qmpRequest struct {
	Execute   string         `json:"execute"`
	Arguments map[string]any `json:"arguments,omitempty"`
	ID        string         `json:"id"`
}

type qmpResponse struct {
	Return json.RawMessage `json:"return"`
	Error  *qmpError       `json:"error"`
	ID     string          `json:"id"`
}

type qmpError struct {
	Class       string `json:"class"`
	Description string `json:"desc"`
}

type QMPClient struct {
	Socket string
	Dial   func(context.Context, string, string) (net.Conn, error)
}

func (client QMPClient) Execute(ctx context.Context, command, id string, arguments map[string]any, result any) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	defer func() {
		if ctx.Err() != nil {
			err = errors.Join(err, ctx.Err())
		}
	}()
	connection, err := client.dial(ctx)
	if err != nil {
		return fmt.Errorf("connect QMP socket: %w", err)
	}
	defer connection.Close()
	stop := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stop()
	deadline := time.Now().Add(30 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err = connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set QMP deadline: %w", err)
	}
	decoder := json.NewDecoder(bufio.NewReader(io.LimitReader(connection, 4<<20)))
	encoder := json.NewEncoder(connection)
	if err = client.negotiate(decoder, encoder); err != nil {
		return err
	}
	request := qmpRequest{Execute: command, Arguments: arguments, ID: id}
	if err = encoder.Encode(request); err != nil {
		return fmt.Errorf("send QMP %s: %w", command, err)
	}
	return decodeQMPResponse(decoder, id, result)
}

func (client QMPClient) dial(ctx context.Context) (net.Conn, error) {
	if client.Socket == "" {
		return nil, errors.New("QMP socket path is required")
	}
	if client.Dial != nil {
		return client.Dial(ctx, "unix", client.Socket)
	}
	return (&net.Dialer{}).DialContext(ctx, "unix", client.Socket)
}

func (client QMPClient) negotiate(decoder *json.Decoder, encoder *json.Encoder) error {
	var greeting map[string]json.RawMessage
	if err := decoder.Decode(&greeting); err != nil {
		return fmt.Errorf("read QMP greeting: %w", err)
	}
	if _, ok := greeting["QMP"]; !ok {
		return errors.New("QMP greeting is missing capabilities")
	}
	if err := encoder.Encode(qmpRequest{Execute: "qmp_capabilities", ID: "capabilities"}); err != nil {
		return fmt.Errorf("enable QMP capabilities: %w", err)
	}
	return decodeQMPResponse(decoder, "capabilities", nil)
}

func decodeQMPResponse(decoder *json.Decoder, id string, result any) error {
	response, err := waitQMPResponse(decoder, id)
	if err != nil {
		return err
	}
	if response.Error != nil {
		return fmt.Errorf("QMP %s failed: %s: %s", id, response.Error.Class, response.Error.Description)
	}
	if len(response.Return) == 0 {
		return errors.New("QMP response has neither result nor error")
	}
	if result == nil {
		return nil
	}
	if err = json.Unmarshal(response.Return, result); err != nil {
		return fmt.Errorf("decode QMP response %s: %w", id, err)
	}
	return nil
}

func waitQMPResponse(decoder *json.Decoder, id string) (qmpResponse, error) {
	for {
		var response qmpResponse
		if err := decoder.Decode(&response); err != nil {
			return qmpResponse{}, fmt.Errorf("read QMP response %s: %w", id, err)
		}
		if response.ID != id {
			continue
		}
		return response, nil
	}
}
