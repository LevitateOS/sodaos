package forgejo

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	upstream "github.com/levitateos/sodaos/internal/forgejo"
)

// ProxyGit forwards a controller-authorized Git request through standard native
// HTTP transport. True signals native authentication rejection, never delivery.
func (p *Provider) ProxyGit(w http.ResponseWriter, r *http.Request, ownerID int64, data []byte, repo upstream.Repository, user upstream.User) bool {
	credential, err := Decode(data, ownerID)
	if err != nil || !validGitProxyIdentity(ownerID, repo, user) {
		http.Error(w, "Git request is unavailable.", http.StatusForbidden)
		return false
	}
	target, err := url.Parse(p.config.Base)
	if err != nil {
		http.Error(w, "Git request is unavailable.", http.StatusBadGateway)
		return false
	}
	rejected := false
	proxy := httputil.ReverseProxy{
		Transport: p.client.HTTP.Transport,
		ErrorLog:  log.New(io.Discard, "", 0),
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			stripGitRequestHeaders(request.Out.Header)
			request.Out.Header["Git-Protocol"] = append([]string(nil), request.In.Header.Values("Git-Protocol")...)
			request.Out.SetBasicAuth(user.Login, credential.Access)
		},
		ModifyResponse: func(response *http.Response) error {
			stripGitResponseHeaders(response.Header)
			stripGitResponseHeaders(response.Trailer)
			if response.StatusCode >= 300 && response.StatusCode < 400 {
				return errors.New("native Git redirect denied")
			}
			rejected = response.StatusCode == http.StatusUnauthorized
			if response.StatusCode >= 400 {
				sanitizeGitResponse(response)
			}
			response.Body = gitResponseBody{ReadCloser: response.Body, response: response}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "Native Git request failed.", http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(w, r)
	return rejected
}

type gitResponseBody struct {
	io.ReadCloser
	response *http.Response
}

func (body gitResponseBody) Read(data []byte) (int, error) {
	n, err := body.ReadCloser.Read(data)
	stripGitResponseHeaders(body.response.Trailer)
	return n, err
}

func validGitProxyIdentity(owner int64, repo upstream.Repository, user upstream.User) bool {
	return user.ID == owner && repo.ID > 0 && safeGitProxyPart(user.Login) && safeGitProxyPart(repo.Owner.Login) && safeGitProxyPart(repo.Name)
}

func safeGitProxyPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && !strings.ContainsAny(value, "/\\:\x00\r\n")
}

func stripGitRequestHeaders(header http.Header) {
	for name := range header {
		lower := strings.ToLower(name)
		if lower == "authorization" || lower == "proxy-authorization" || lower == "cookie" || lower == "forwarded" || strings.HasPrefix(lower, "x-forwarded-") || strings.HasPrefix(lower, "x-soda-") {
			delete(header, name)
		}
	}
}

func stripGitResponseHeaders(header http.Header) {
	for _, name := range []string{"Set-Cookie", "Authorization", "Proxy-Authenticate", "WWW-Authenticate", "Location"} {
		header.Del(name)
	}
}

func sanitizeGitResponse(response *http.Response) {
	_ = response.Body.Close()
	message := "Native Git request failed.\n"
	response.Body = io.NopCloser(strings.NewReader(message))
	response.ContentLength = int64(len(message))
	response.Header.Set("Content-Type", "text/plain; charset=utf-8")
	response.Header.Set("Content-Length", strconv.Itoa(len(message)))
	response.Header.Del("Content-Encoding")
	response.Header.Del("Content-Range")
	response.Trailer = nil
}
