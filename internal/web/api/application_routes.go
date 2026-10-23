package api

import "net/http"

func extensionApplicationRoute(parts []string, method string) bool {
	if len(parts) == 1 {
		return method == http.MethodGet || method == http.MethodPost
	}
	if parts[1] == "setups" {
		return (len(parts) == 2 && (method == http.MethodGet || method == http.MethodPost)) || (len(parts) == 4 && extensionProductID(parts[2]) && parts[3] == "actions" && method == http.MethodPost)
	}
	if !extensionProductID(parts[1]) {
		return false
	}
	if len(parts) == 2 {
		return method == http.MethodGet
	}
	if len(parts) == 3 && parts[2] == "snapshots" {
		return method == http.MethodGet || method == http.MethodPost
	}
	return len(parts) == 3 && (parts[2] == "actions" || parts[2] == "project" || parts[2] == "restores") && method == http.MethodPost
}
