package utils

import (
	"log/slog"
	"net/http"
	"time"
)

// LogLevel controls structured log severity.
type LogLevel = slog.Level

// AccessLog wraps an HTTP handler and writes one structured log entry per request.
func AccessLog(next http.Handler, now func() time.Time, level LogLevel) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		startedAt := now()
		recorder := &statusRecorder{ResponseWriter: response, status: http.StatusOK}

		next.ServeHTTP(recorder, request)

		slog.Log(request.Context(), level, "http access",
			"method", request.Method,
			"path", request.URL.Path,
			"query", request.URL.RawQuery,
			"status", recorder.status,
			"bytes", recorder.bytes,
			"duration", now().Sub(startedAt).String(),
			"remote_addr", request.RemoteAddr,
			"user_agent", request.UserAgent(),
		)
	})
}

// LogHTTPError writes a structured error log for an HTTP error response.
func LogHTTPError(request *http.Request, status int, message string) {
	slog.ErrorContext(request.Context(), "http error",
		"method", request.Method,
		"path", request.URL.Path,
		"query", request.URL.RawQuery,
		"status", status,
		"error", message,
		"remote_addr", request.RemoteAddr,
		"user_agent", request.UserAgent(),
	)
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if recorder.wroteHeader {
		return
	}
	recorder.status = status
	recorder.wroteHeader = true
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(content []byte) (int, error) {
	if !recorder.wroteHeader {
		recorder.wroteHeader = true
	}
	written, err := recorder.ResponseWriter.Write(content)
	recorder.bytes += written
	return written, err
}
