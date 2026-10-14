package api

import "net/http"

func extensionApplicationRoute(parts []string, method string) bool {
	if len(parts) == 1 {
		return method == http.MethodGet || method == http.MethodPost
	}
	if !extensionProductID(parts[1]) {
		return false
	}
	if len(parts) == 2 {
		return method == http.MethodGet
	}
	return len(parts) == 3 && parts[2] == "actions" && method == http.MethodPost
}
