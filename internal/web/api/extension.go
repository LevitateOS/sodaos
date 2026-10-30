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
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/web/auth"
)

var (
	errExtensionResponseTooLarge = errors.New("soda response exceeds the API limit")
	errExtensionResponseInvalid  = errors.New("soda response could not be read")
)

const extensionResponseHeaderTimeout = stopOperationTimeout + time.Minute

// Extension prepares the initial read/preferences bridge over a private Unix
// socket. The receiving Soda listener must verify the native admission itself;
// this transport must never point at the public dashboard browser listener.
func Extension(socket string) (http.Handler, func()) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", socket)
		},
		// An admitted mutation may outlive the browser/Fountain request and
		// is bounded by its detached operation context. Keep this transport
		// open long enough to return that operation's recorded outcome.
		ResponseHeaderTimeout: extensionResponseHeaderTimeout,
	}
	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite:   rewriteExtensionRequest,
		ModifyResponse: func(response *http.Response) error {
			if response.StatusCode != http.StatusSwitchingProtocols {
				if err := boundExtensionResponse(response); err != nil {
					return err
				}
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
			logExtensionDenial(r, "authority", err, authority.Contribution)
			auth.JSONError(w, http.StatusForbidden, "native_authority_unavailable", "Current native authority is required.")
			return
		}
		if !auth.ExtensionContribution(authority.ExtensionID, authority.Contribution) {
			logExtensionDenial(r, "contribution", nil, authority.Contribution)
			auth.JSONError(w, http.StatusForbidden, "invalid_contribution", "Soda contribution is unavailable.")
			return
		}
		if !extensionTerminalStreamRoute(r) && !extensionFactoryOutputStreamRoute(r) {
			r.Body = http.MaxBytesReader(w, r.Body, auth.APIBodyLimit)
		}
		proxy.ServeHTTP(w, r)
	})
	return handler, transport.CloseIdleConnections
}

func boundExtensionResponse(response *http.Response) error {
	if response.Body == nil {
		return nil
	}
	limit := auth.APIBodyLimit
	if response.Request != nil && response.Request.Method == http.MethodGet && extensionArtifactRoute(strings.TrimPrefix(response.Request.URL.Path, "/api")) {
		limit = factory.MaxFeedArtifactBytes
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	closeErr := response.Body.Close()
	if err != nil || closeErr != nil {
		return fmt.Errorf("%w: response body read failed", errExtensionResponseInvalid)
	}
	if len(body) > limit {
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
	if !extensionCleanRoute(r) {
		return false
	}
	return extensionSessionRoute(r) || extensionMeRoute(r) || extensionTerminalRoute(r) || extensionProductRoute(r)
}

func extensionCleanRoute(r *http.Request) bool {
	if r.URL.RawPath != "" || r.URL.ForceQuery ||
		strings.Contains(r.URL.Path, "\\") || path.Clean(r.URL.Path) != r.URL.Path {
		return false
	}
	return r.URL.RawQuery == "" || extensionQueryRoute(r.URL.Path)
}

func extensionSessionRoute(r *http.Request) bool {
	return r.URL.Path == "/session" && r.Method == http.MethodGet
}

func extensionMeRoute(r *http.Request) bool {
	switch r.URL.Path {
	case "/me/preferences":
		return r.Method == http.MethodGet || r.Method == http.MethodPatch
	case "/me/development-keys":
		return r.Method == http.MethodGet || r.Method == http.MethodPost
	case "/me/forgejo-keys":
		return r.Method == http.MethodGet
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	return extensionDevelopmentKeyRoute(parts, r.Method)
}

func extensionDevelopmentKeyRoute(parts []string, method string) bool {
	return len(parts) == 3 && parts[0] == "me" && parts[1] == "development-keys" && parts[2] != "" &&
		method == http.MethodDelete
}

func extensionQueryRoute(route string) bool {
	switch route {
	case "/repositories", "/environments", "/me/forgejo-keys", "/factory/intents", "/factory/feed", "/factory/library", "/factory/library/item", "/spaces":
		return true
	default:
		if extensionArtifactRoute(route) {
			return true
		}
		parts := strings.Split(strings.TrimPrefix(route, "/"), "/")
		if len(parts) == 3 && parts[0] == "repositories" && extensionProductID(parts[1]) && parts[2] == "application-builds" {
			return true
		}
		if len(parts) == 5 && parts[0] == "factory" && parts[1] == "feed" && parts[2] == "posts" && parts[3] != "" && parts[4] == "replies" {
			return true
		}
		if len(parts) == 6 && parts[0] == "repositories" && extensionProductID(parts[1]) &&
			parts[2] == "factory" && parts[3] == "issues" && extensionProductID(parts[4]) {
			return parts[5] == "acceptance-preview" || parts[5] == "plan"
		}
		return len(parts) == 4 && parts[0] == "factory" && parts[1] == "intents" &&
			parts[2] != "" && (parts[3] == "revisions" || parts[3] == "review")
	}
}

// extensionProductRoute admits only the current Soda product API paths. The
// private listener applies the same routes and native authority independently.
func extensionProductRoute(r *http.Request) bool {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return false
	}
	switch parts[0] {
	case "repositories":
		return extensionRepositoryRoute(parts, r.Method)
	case "factory":
		if admitted, handled := extensionFactoryIntentsRoute(parts, r.Method); handled {
			return admitted
		}
		return extensionFactoryRoute(parts, r.Method)
	case "spaces":
		return len(parts) == 1 && r.Method == http.MethodGet
	case "environments":
		return extensionEnvironmentRoute(parts, r.Method)
	case "applications":
		return extensionApplicationRoute(parts, r.Method)
	case "identity":
		return extensionIdentityRoute(parts, r.Method)
	case "settings":
		return extensionSettingsRoute(parts, r.Method)
	}
	return false
}

func extensionProductID(value string) bool {
	return value != "" && value != "." && value != ".."
}

func extensionFactoryIntentsRoute(parts []string, method string) (admitted, handled bool) {
	if admitted, handled := extensionIntentsRevisionRoute(parts, method); handled {
		return admitted, true
	}
	return extensionIntentsMutationRoute(parts, method)
}

func extensionIntentsRevisionRoute(parts []string, method string) (admitted, handled bool) {
	if len(parts) == 4 && parts[1] == "intents" && parts[2] != "" && parts[3] == "verification-allowance" {
		return method == http.MethodGet || method == http.MethodPost, true
	}
	if len(parts) == 4 && parts[1] == "intents" && parts[2] != "" && parts[3] == "plan" {
		return method == http.MethodGet || method == http.MethodPost, true
	}
	if len(parts) == 4 && parts[1] == "intents" && parts[2] != "" && (parts[3] == "generation" || parts[3] == "history" || parts[3] == "overview" || parts[3] == "revisions" || parts[3] == "review") {
		return method == http.MethodGet, true
	}
	if len(parts) == 3 && parts[1] == "intents" && parts[2] != "" {
		return method == http.MethodGet, true
	}
	return false, false
}

func extensionIntentsMutationRoute(parts []string, method string) (admitted, handled bool) {
	if len(parts) == 5 && parts[1] == "intents" && parts[2] != "" && parts[3] == "plan" && parts[4] == "review" {
		return method == http.MethodPost, true
	}
	if len(parts) == 2 && parts[1] == "departments" {
		return method == http.MethodGet || method == http.MethodPost, true
	}
	if len(parts) == 2 && parts[1] == "intents" {
		return method == http.MethodPost || method == http.MethodGet, true
	}
	if len(parts) == 4 && parts[1] == "intents" && parts[2] != "" && (parts[3] == "explore" || parts[3] == "generate" || parts[3] == "plan") {
		return method == http.MethodPost, true
	}
	return false, false
}

func extensionRepositoryRoute(parts []string, method string) bool {
	if len(parts) >= 3 && extensionProductID(parts[1]) && parts[2] == "application-builds" {
		return len(parts) == 3 && (method == http.MethodGet || method == http.MethodPost) ||
			len(parts) == 5 && extensionProductID(parts[3]) && parts[4] == "setup" && method == http.MethodPost
	}
	switch len(parts) {
	case 1:
		return method == http.MethodGet
	case 3:
		return extensionRepositoryProfileRoute(parts, method)
	case 4:
		return extensionRepositoryFactoryAction(parts, method)
	case 5:
		return extensionRepositoryFactoryResource(parts, method)
	case 6:
		return extensionRepositoryFactoryItem(parts, method)
	}
	return false
}

func extensionRepositoryProfileRoute(parts []string, method string) bool {
	if !extensionProductID(parts[1]) {
		return false
	}
	if parts[2] == "profiles" || parts[2] == "tailnet-options" {
		return method == http.MethodGet
	}
	return parts[2] == "factory" && method == http.MethodGet
}

func extensionRepositoryFactoryAction(parts []string, method string) bool {
	if !extensionProductID(parts[1]) || parts[2] != "factory" {
		return false
	}
	switch parts[3] {
	case "access":
		return method == http.MethodGet
	case "activity":
		return method == http.MethodGet || method == http.MethodPut
	case "policy", "operator-grant", "environment-grant":
		return method == http.MethodPut
	case "repair":
		return method == http.MethodGet || method == http.MethodPut
	case "actions":
		return method == http.MethodPost
	}
	return false
}

func extensionRepositoryFactoryResource(parts []string, method string) bool {
	if !extensionProductID(parts[1]) || parts[2] != "factory" || !extensionProductID(parts[4]) {
		return false
	}
	switch parts[3] {
	case "sponsorships":
		return method == http.MethodPut
	case "issues":
		return method == http.MethodGet
	case "lifecycle":
		return method == http.MethodGet || method == http.MethodPut
	}
	return false
}

func extensionRepositoryFactoryItem(parts []string, method string) bool {
	if !extensionProductID(parts[1]) || parts[2] != "factory" || !extensionProductID(parts[4]) {
		return false
	}
	if parts[3] == "lifecycle" {
		return parts[5] == "actions" && method == http.MethodGet
	}
	if parts[3] == "issues" {
		return parts[5] == "acceptances" && method == http.MethodPost ||
			parts[5] == "acceptance-preview" && method == http.MethodGet ||
			parts[5] == "assignment" && method == http.MethodGet ||
			parts[5] == "plan" && method == http.MethodGet ||
			parts[5] == "withdrawal" && method == http.MethodPost
	}
	return false
}

func extensionFactoryRoute(parts []string, method string) bool {
	if len(parts) >= 2 && parts[1] == "library" {
		return method == http.MethodGet && (len(parts) == 2 || len(parts) == 3 && parts[2] == "item")
	}
	if len(parts) == 3 && parts[1] == "artifacts" {
		return method == http.MethodGet && factory.ValidID(parts[2])
	}
	if len(parts) >= 2 && parts[1] == "feed" {
		switch len(parts) {
		case 2:
			return method == http.MethodGet
		case 3:
			return parts[2] == "subscription" && (method == http.MethodGet || method == http.MethodPut) ||
				parts[2] == "posts" && method == http.MethodPost
		case 4:
			return parts[2] == "posts" && extensionProductID(parts[3]) && method == http.MethodGet
		case 5:
			return parts[2] == "posts" && extensionProductID(parts[3]) &&
				(parts[4] == "replies" && method == http.MethodGet ||
					(parts[4] == "retry" || parts[4] == "review") && method == http.MethodPost)
		}
		return false
	}
	switch len(parts) {
	case 2:
		return parts[1] == "capacity" && method == http.MethodPut
	case 3:
		if parts[1] == "commands" && extensionProductID(parts[2]) {
			return method == http.MethodGet
		}
		return parts[1] == "runs" && extensionProductID(parts[2]) && method == http.MethodGet
	case 4:
		if parts[1] != "runs" || !extensionProductID(parts[2]) {
			return false
		}
		return parts[3] == "actions" && method == http.MethodPost ||
			parts[3] == "output" && method == http.MethodGet
	}
	return false
}

func extensionArtifactRoute(route string) bool {
	parts := strings.Split(strings.TrimPrefix(route, "/"), "/")
	return len(parts) == 3 && parts[0] == "factory" && parts[1] == "artifacts" && factory.ValidID(parts[2])
}

func extensionEnvironmentRoute(parts []string, method string) bool {
	switch len(parts) {
	case 1:
		return method == http.MethodGet || method == http.MethodPost
	case 2:
		return extensionProductID(parts[1]) && method == http.MethodGet
	case 3:
		return extensionProductID(parts[1]) && extensionEnvironmentActionRoute(parts[2], method)
	case 4:
		return extensionEnvironmentIdentityPath(parts, method) || extensionEnvironmentPreparationPath(parts, method)
	case 5:
		_, member := auth.PositiveID(parts[3])
		return extensionProductID(parts[1]) && parts[2] == "members" && member && parts[4] == "access" && (method == http.MethodGet || method == http.MethodPost)
	}
	return false
}

func extensionEnvironmentPreparationPath(parts []string, method string) bool {
	return extensionProductID(parts[1]) && parts[2] == "preparation" &&
		(parts[3] == "acceptances" || parts[3] == "actions") && method == http.MethodPost
}

func extensionEnvironmentIdentityPath(parts []string, method string) bool {
	return extensionProductID(parts[1]) && parts[2] == "identity" && extensionEnvironmentIdentityRoute(parts[3], method)
}

func extensionEnvironmentActionRoute(action, method string) bool {
	switch action {
	case "join":
		return method == http.MethodPost
	case "tailnet":
		return method == http.MethodGet || method == http.MethodPost
	case "members", "connection", "os":
		return method == http.MethodGet
	case "lifecycle", "access-keys", "preparation":
		return method == http.MethodGet || method == http.MethodPost
	}
	return false
}

func extensionEnvironmentIdentityRoute(action, method string) bool {
	switch action {
	case "launch", "grants":
		return method == http.MethodPost
	case "connections":
		return method == http.MethodGet
	}
	return false
}

func extensionIdentityRoute(parts []string, method string) bool {
	if len(parts) < 2 {
		return false
	}
	switch parts[1] {
	case "jev":
		if len(parts) == 2 {
			return method == http.MethodGet || method == http.MethodPost
		}
		return len(parts) == 3 && parts[2] == "disconnect" && method == http.MethodPost
	case "connections":
		return extensionIdentityConnectionsRoute(parts, method)
	case "enrollments":
		return extensionIdentityEnrollmentsRoute(parts, method)
	case "grants":
		return extensionIdentityActionRoute(parts, method, "revoke")
	case "leases":
		return extensionIdentityActionRoute(parts, method, "end")
	}
	return false
}

func extensionIdentityActionRoute(parts []string, method, action string) bool {
	return len(parts) == 4 && extensionProductID(parts[2]) && parts[3] == action && method == http.MethodPost
}

func extensionIdentityConnectionsRoute(parts []string, method string) bool {
	if len(parts) == 2 {
		return method == http.MethodGet
	}
	if len(parts) != 4 || !extensionProductID(parts[2]) {
		return false
	}
	switch parts[3] {
	case "grants", "leases":
		return method == http.MethodGet
	case "revoke":
		return method == http.MethodPost
	}
	return false
}

func extensionIdentityEnrollmentsRoute(parts []string, method string) bool {
	switch len(parts) {
	case 2:
		return method == http.MethodPost || method == http.MethodGet
	case 3:
		return extensionProductID(parts[2]) && method == http.MethodGet
	case 4:
		return extensionProductID(parts[2]) && parts[3] == "cancel" && method == http.MethodPost
	}
	return false
}

func extensionSettingsRoute(parts []string, method string) bool {
	if len(parts) < 2 {
		return false
	}
	switch parts[1] {
	case "tailnet":
		return extensionTailnetSettingsRoute(parts, method)
	}
	return false
}

func extensionTailnetSettingsRoute(parts []string, method string) bool {
	if len(parts) == 2 {
		return method == http.MethodGet
	}
	return len(parts) == 3 && (parts[2] == "host" || parts[2] == "enrollment") && method == http.MethodPost
}

func extensionTerminalStreamRoute(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	return len(parts) == 4 && parts[1] == "environments" && parts[2] != "" && parts[3] == "terminal" && r.Method == http.MethodGet
}

// extensionFactoryOutputStreamRoute admits the read-only factory output
// stream through the proxy with its WebSocket upgrade headers. Proxy paths
// carry no /api prefix; the private service route is registered separately.
func extensionFactoryOutputStreamRoute(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	return len(parts) == 5 && parts[1] == "factory" && parts[2] == "runs" && parts[3] != "" && parts[4] == "output" && r.Method == http.MethodGet
}

func extensionTerminalRoute(r *http.Request) bool {
	if extensionTerminalStreamRoute(r) {
		return true
	}
	parts := strings.Split(r.URL.Path, "/")
	return extensionTerminalSessionRoute(parts, r.Method)
}

func extensionTerminalSessionRoute(parts []string, method string) bool {
	if len(parts) < 4 || parts[1] != "environments" || parts[2] == "" || parts[3] != "terminal-sessions" {
		return false
	}
	return (len(parts) == 4 && method == http.MethodPost) ||
		(len(parts) == 5 && parts[4] != "" && terminalControlMethod(method))
}

func terminalControlMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodPost
}

func rewriteExtensionRequest(request *httputil.ProxyRequest) {
	request.Out.URL.Scheme = "http"
	request.Out.URL.Host = "soda-extension-service"
	request.Out.URL.Path = "/api" + request.In.URL.Path
	request.Out.Host = "soda-extension-service"
	request.Out.Header = make(http.Header)
	for _, name := range []string{
		"Accept", "Content-Type", "Origin", "Sec-Fetch-Site", "Sec-Fetch-Mode", "Sec-Fetch-Dest", "X-CSRF-Token", extensions.SessionGenerationHeader,
		extensions.ContextHeader, extensions.AdmissionHeader,
	} {
		for _, value := range request.In.Header.Values(name) {
			request.Out.Header.Add(name, value)
		}
	}
	if extensionTerminalStreamRoute(request.In) || extensionFactoryOutputStreamRoute(request.In) {
		for _, name := range []string{"Connection", "Upgrade", "Sec-WebSocket-Key", "Sec-WebSocket-Version"} {
			for _, value := range request.In.Header.Values(name) {
				request.Out.Header.Add(name, value)
			}
		}
	}
}
