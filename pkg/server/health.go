package server

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jeks313/go-mongo-slow-queries/pkg/health"
)

// Health sets up the default health router
func Health(r *mux.Router, route string, dependencies ...*health.Dependency) {
	health.RegisterDependencies(dependencies...)
	health.Serve()

	r.PathPrefix(route).Handler(health.WebHandler())
}
