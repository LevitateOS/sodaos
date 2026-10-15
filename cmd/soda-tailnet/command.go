package main

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"time"

	"github.com/levitateos/sodaos/internal/tailnet"
)

func execute(ctx context.Context, output io.Writer, status func(context.Context) (tailnet.SettingsView, error)) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	current, statusErr := status(ctx)
	_, err := io.WriteString(output, enrollmentMessage(current, statusErr))
	return err
}

func enrollmentMessage(view tailnet.SettingsView, statusErr error) string {
	guidance := "Administrator: open Admin → Extensions → Tailnet → Settings to connect this machine.\n"
	if statusErr != nil || view.HostUnavailable || view.Host == nil {
		return "\nTailscale status is unavailable.\n" + guidance
	}
	status := *view.Host
	if status.Expired {
		return "\nTailscale authentication has expired.\n" + guidance
	}
	if status.State != "Running" {
		return "\nTailscale: " + connectionDescription(status) + ".\n" + guidance
	}
	message := "\nTailscale: connected.\n"
	if status.DNSName != "" {
		message += fmt.Sprintf("Tailnet identity: %s\n", status.DNSName)
	}
	if host := endpointHost(status); host != "" {
		return message + fmt.Sprintf("Tailnet endpoint: %s\nBrowser services use their configured HTTPS origins; Cockpit listens on all interfaces.\n", host)
	}
	return message + "Tailnet service addresses are unavailable.\n"
}

func connectionDescription(status tailnet.HostView) string {
	descriptions := map[string]string{
		"NeedsLogin": "not signed in", "NeedsMachineAuth": "waiting for Tailnet administrator approval",
		"Stopped": "disconnected", "Starting": "connecting", "NoState": "starting",
	}
	if description, ok := descriptions[status.State]; ok {
		return description
	}
	return status.State
}

func endpointHost(status tailnet.HostView) string {
	if status.MagicDNSEnabled && status.DNSName != "" {
		return status.DNSName
	}
	for _, value := range status.Addresses {
		if address, err := netip.ParseAddr(value); err == nil && address.Is4() {
			return address.String()
		}
	}
	return ""
}
