//go:build linux

// soda-muse-maintain installs public Muse tools and restores a live project interface.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/platform"
	"github.com/levitateos/sodaos/internal/project"
)

type options struct {
	config, project, tools string
	bindOnly               bool
	gitOnly                bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "soda-muse-maintain:", err)
		os.Exit(1)
	}
}

func parse(args []string) (options, error) {
	var o options
	flags := flag.NewFlagSet("soda-muse-maintain", flag.ContinueOnError)
	flags.StringVar(&o.config, "config", "/etc/soda/host.json", "operator host configuration")
	flags.StringVar(&o.project, "project", "", "exact project identity")
	flags.StringVar(&o.tools, "tools", filepath.Clean(filepath.Join(platform.Libexec, "../../share/soda/muse-tools")), "installed public tool directory")
	flags.BoolVar(&o.gitOnly, "git-only", false, "maintain only the native Git launch interface and public helper")
	flags.BoolVar(&o.bindOnly, "bind-only", false, "restore only the launch interface")
	if err := flags.Parse(args); err != nil {
		return o, err
	}
	if flags.NArg() != 0 || !project.ValidID(o.project) || !filepath.IsAbs(o.config) || !filepath.IsAbs(o.tools) {
		return o, errors.New("explicit project and absolute maintenance paths required")
	}
	return o, nil
}

func run() error {
	if os.Geteuid() != 0 {
		return errors.New("project maintenance requires root")
	}
	o, err := parse(os.Args[1:])
	if err != nil {
		return err
	}
	c, err := host.LoadConfig(o.config)
	if err != nil {
		return err
	}
	if o.gitOnly {
		return maintainGit(o, c)
	}
	if c.MuseSHA256 == "" {
		if o.bindOnly {
			return nil
		}
		return errors.New("muse project runtime is disabled")
	}
	return maintain(o, c)
}

func maintain(o options, c host.Config) error {
	sources, err := loadTools(o.tools, c.MuseSHA256, c.MuseVersion)
	if err != nil {
		return err
	}
	defer closeTools(sources)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	target, err := waitProject(ctx, o.project)
	if err != nil {
		return err
	}
	if !o.bindOnly {
		if err := ensureSystemBus(ctx, target); err != nil {
			return err
		}
		if err := stageTools(ctx, target, sources); err != nil {
			return err
		}
	}
	if err := prepareInterface(ctx, target); err != nil {
		return err
	}
	return attachInterface(ctx, target, c.MuseSocket)
}
