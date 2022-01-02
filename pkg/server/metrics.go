package server

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics installs the prometheus handler for the metrics endpoint
func Metrics(r *mux.Router, route string) {
	if route == "" {
		route = "/metrics"
	}
	r.Handle(route, promhttp.Handler())
}
