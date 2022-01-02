package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Redirect takes one path and redirects it to another
func Redirect(r *mux.Router, from, to string) {
	r.Handle(from, redirectHandler(to))
}

func redirectHandler(path string) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, path, http.StatusPermanentRedirect)
	})
}
