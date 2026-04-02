package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	responseData struct {
		status int
		size   int
	}

	responseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func WithLogging(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := &responseWriter{ResponseWriter: w,
				responseData: &responseData{
					status: 0,
					size:   0,
				}}

			h.ServeHTTP(lw, r)

			duration := time.Since(start)

			logger.Infow("HTTP request",
				"method", r.Method,
				"uri", r.RequestURI,
				"status", lw.responseData.status,
				"duration", duration,
				"size", lw.responseData.size,
				"content_type", r.Header.Get("Content-Type"),
				"content_encoding", r.Header.Get("Content-Encoding"),
				"accept_encoding", r.Header.Get("Accept-Encoding"),
				"response_header", lw.ResponseWriter.Header(),
			)
		})
	}
}
