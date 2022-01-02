package server

import (
	"container/ring"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
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

// Log sets up default http logging
func Log(r *mux.Router, log zerolog.Logger) {
	// Install the logger handler with default output on the console
	r.Use(hlog.NewHandler(log))
	// Install some provided extra handler to set some request's context fields.
	// Thanks to those handler, all our logs will come with some pre-populated fields.
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("request")
	}))
	r.Use(hlog.RemoteAddrHandler("ip"))
	r.Use(hlog.UserAgentHandler("user_agent"))
	r.Use(hlog.RefererHandler("referer"))
	r.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
}
