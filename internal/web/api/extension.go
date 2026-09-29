package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"path"
	"strconv"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/web/auth"
)

var (
	errExtensionResponseTooLarge = errors.New("soda response exceeds the API limit")
	errExtensionResponseInvalid  = errors.New("soda response could not be read")
)

// Extension prepares the initial read/preferences bridge over a private Unix
// socket. The receiving Soda listener must verify the native admission itself;
// this transport must never point at the public dashboard browser listener.
func Extension(socket string) (http.Handler, func()) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", socket)
		},
		ResponseHeaderTimeout: 10 * time.Second,
	}
	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite:   rewriteExtensionRequest,
		ModifyResponse: func(response *http.Response) error {
			if err := boundExtensionResponse(response); err != nil {
				return err
			}
			response.Header.Del("Set-Cookie")
			response.Header.Del(extensions.ContextHeader)
			response.Header.Del(extensions.AdmissionHeader)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			if errors.Is(err, errExtensionResponseTooLarge) || errors.Is(err, errExtensionResponseInvalid) {
				auth.JSONError(w, http.StatusBadGateway, "invalid_soda_response", "Soda returned an invalid response.")
				return
			}
			auth.JSONError(w, http.StatusServiceUnavailable, "soda_unavailable", "Soda service is unavailable.")
		},
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !extensionRoute(r) {
			http.NotFound(w, r)
			return
		}
		authority, err := auth.ExtensionAuthority(r)
		if err != nil {
			auth.JSONError(w, http.StatusForbidden, "native_authority_unavailable", "Current native authority is required.")
			return
		}
		if !auth.ExtensionContribution(authority.Contribution) {
			auth.JSONError(w, http.StatusForbidden, "invalid_contribution", "Soda contribution is unavailable.")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, auth.APIBodyLimit)
		proxy.ServeHTTP(w, r)
	})
	return handler, transport.CloseIdleConnections
}

func boundExtensionResponse(response *http.Response) error {
	if response.Body == nil {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(auth.APIBodyLimit)+1))
	closeErr := response.Body.Close()
	if err != nil || closeErr != nil {
		return fmt.Errorf("%w: response body read failed", errExtensionResponseInvalid)
	}
	if len(body) > auth.APIBodyLimit {
		return errExtensionResponseTooLarge
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Length", strconv.Itoa(len(body)))
	response.Header.Del("Transfer-Encoding")
	response.TransferEncoding = nil
	return nil
}

func extensionRoute(r *http.Request) bool {
	if r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		strings.Contains(r.URL.Path, "\\") || path.Clean(r.URL.Path) != r.URL.Path {
		return false
	}
	switch r.URL.Path {
	case "/session":
		return r.Method == http.MethodGet
	case "/me/preferences":
		return r.Method == http.MethodGet || r.Method == http.MethodPatch
	default:
		return false
	}
}

func rewriteExtensionRequest(request *httputil.ProxyRequest) {
	request.Out.URL.Scheme = "http"
	request.Out.URL.Host = "soda-extension-service"
	request.Out.URL.Path = "/api" + request.In.URL.Path
	request.Out.Host = "soda-extension-service"
	request.Out.Header = make(http.Header)
	for _, name := range []string{
		"Accept", "Content-Type", "Origin", "Sec-Fetch-Site", "X-CSRF-Token",
		extensions.ContextHeader, extensions.AdmissionHeader,
	} {
		for _, value := range request.In.Header.Values(name) {
			request.Out.Header.Add(name, value)
		}
	}
}
