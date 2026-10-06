package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"

	extensions "forgejo.org/extension-sdk"

	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

// maxBackgroundStatusBody bounds a captured dispatch status body: refusal
// codes are short lines, never bulk content.
const maxBackgroundStatusBody = 1024

func checkBackgroundLookup(b *ServiceBackground, operationID string, lookup extensions.OperationLookup) error {
	b.mu.Lock()
	pinned := b.pinned
	b.mu.Unlock()
	if pinned == "" || lookup.InstallationID != pinned || lookup.OperationID != operationID || lookup.Status == "" {
		return errors.New("invalid background response")
	}
	if lookup.Status == extensions.BackgroundOutcomeNotObserved {
		if lookup.Record != nil {
			return errors.New("invalid background response")
		}
		return nil
	}
	if lookup.Record == nil {
		return errors.New("invalid background response")
	}
	return checkBackgroundRecord(b, operationID, *lookup.Record)
}

// call posts one background request under the current admission. A 401
// rebinds once and retries once: either the admission was revoked and
// the retry succeeds, or the retry's verdict stands.
func (b *ServiceBackground) call(ctx context.Context, path string, payload any, target any) error {
	admission, err := b.admissionForCall(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := b.post(ctx, path, data, admission, target); err != nil {
		var status *forgejopublish.StatusError
		if !errors.As(err, &status) || status.Status != http.StatusUnauthorized {
			return err
		}
		rebound, rebindErr := b.bootstrap(ctx, true)
		if rebindErr != nil {
			return rebindErr
		}
		if rebound == admission {
			return err
		}
		return b.post(ctx, path, data, rebound, target)
	}
	return nil
}

func (b *ServiceBackground) post(ctx context.Context, path string, data []byte, admission string, target any) error {
	b.mu.Lock()
	socket := b.socket
	b.mu.Unlock()
	transport := &http.Transport{DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "unix", socket)
		if err != nil {
			return nil, err
		}
		return b.verifiedPeer(conn)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://extensionHost"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(extensions.AdmissionHeader, admission)
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("background request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &forgejopublish.StatusError{Status: response.StatusCode, Body: readStatusBody(response.Body)}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBackgroundBodyBytes+1))
	if err != nil || len(body) > maxBackgroundBodyBytes {
		return errors.New("invalid background response")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid background response")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("invalid background response")
	}
	return nil
}

func readStatusBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxBackgroundStatusBody+1))
	if err != nil || len(data) > maxBackgroundStatusBody {
		return ""
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	return string(bytes.TrimSpace(line))
}
