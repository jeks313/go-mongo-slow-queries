package server

import (
	"net/http"
	"net/http/pprof"
	"path"

	"github.com/gorilla/mux"
)

// Profiling installs the go default profiling endpoints
func Profiling(r *mux.Router, route string) {
	// add the profiling endpoints
	r.Handle(path.Join(route, "/"), http.HandlerFunc(pprof.Index))
	r.Handle(path.Join(route, "/profile"), http.HandlerFunc(pprof.Profile))
	r.Handle(path.Join(route, "/cmdline"), http.HandlerFunc(pprof.Cmdline))
	r.Handle(path.Join(route, "/trace"), http.HandlerFunc(pprof.Trace))
	r.Handle(path.Join(route, "/symbol"), http.HandlerFunc(pprof.Symbol))

	r.Handle(path.Join(route, "mutex"), pprof.Handler("mutex"))
	r.Handle(path.Join(route, "heap"), pprof.Handler("heap"))
	r.Handle(path.Join(route, "goroutine"), pprof.Handler("goroutine"))
	r.Handle(path.Join(route, "block"), pprof.Handler("block"))
	r.Handle(path.Join(route, "threadcreate"), pprof.Handler("threadcreate"))
}
