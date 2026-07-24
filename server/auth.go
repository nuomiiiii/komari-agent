package server

import "net/http"

func agentAuthorizationHeader(token string) http.Header {
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+token)
	return headers
}

func authorizeAgentRequest(request *http.Request, token string) {
	request.Header.Set("Authorization", "Bearer "+token)
}
