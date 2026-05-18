package utils

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

// WriteJSON writes a JSON response with a status code.
func WriteJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

// WriteJSONError logs and writes a JSON error response with a status code.
func WriteJSONError(response http.ResponseWriter, request *http.Request, status int, message string) {
	LogHTTPError(request, status, message)
	WriteJSON(response, status, errorResponse{Error: message})
}
