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
}

func NewStatusReponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (sw *statusResponseWriter) WriteHeader(statusCode int) {
	sw.statusCode = statusCode
	sw.ResponseWriter.WriteHeader(statusCode)
}

// RequestLoggerMiddleware takes care of logging all requests
func RequestLoggerMiddleware(r *mux.Router) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			sw := NewStatusReponseWriter(w)
			defer func() {
				slog.Info(req.URL.RawQuery,
					"method", req.Method,
					"duration", time.Since(start),
					"host", req.Host,
					"path", req.URL.Path,
					"status", sw.statusCode)
			}()
			next.ServeHTTP(sw, req)
		})
	}
}

// Log sets up default http logging
func Log(r *mux.Router) {
	return RequestLoggerMiddleware(r)
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
