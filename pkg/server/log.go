package server

import (
	"container/ring"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

// LogBuffer buffers the last number of log lines for the log output handler
type LogBuffer struct {
	buf *ring.Ring
}

// Run is the zerolog hook to install
func (h *LogBuffer) Run(e *zerolog.Event, level zerolog.Level, msg string) {
}

// LogHandler sets up a log circular buffer and serves this on the given router
func LogHandler(r *mux.Router, route string, length int) {
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
	length     int
}

// NewStatusReponseWriter creates a new response writer that is used to store
// the status code of the response for later logging in the log middleware
func NewStatusReponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (s *statusResponseWriter) Write(data []byte) (n int, err error) {
	s.length += n
	return s.ResponseWriter.Write(data)
}

func (s *statusResponseWriter) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

// RequestLoggerMiddleware takes care of logging all requests
func RequestLoggerMiddleware(r *mux.Router) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			sw := NewStatusReponseWriter(w)
			defer func() {
				slog.Info("request",
					"method", req.Method,
					"duration_seconds", time.Since(start).Seconds(),
					"url", req.URL.String(),
					"path", req.URL.Path,
					"size", sw.length,
					"status", sw.statusCode)
			}()
			next.ServeHTTP(sw, req)
		})
	}
}

// Log sets up default http logging
func Log(r *mux.Router) {
	r.Use(RequestLoggerMiddleware(r))
	/*
		r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			hlog.FromRequest(r).Info().
				Str("method", r.Method).
				Str("url", r.URL.String()).
				Int("status", status).
				Int("size", size).
				Dur("duration_ms", duration).
				Msg("request")
		}))
		r.Use(hlog.RemoteAddrHandler("ip"))
		r.Use(hlog.UserAgentHandler("user_agent"))
		r.Use(hlog.RefererHandler("referer"))
		r.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	*/
}
