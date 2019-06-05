package mux

import (
	"net/http"
)

// defCodeWriter is an http.ResponseWriter that writes the given status code by
// default instead of always defaulting to a 200.
type defCodeWriter struct {
	http.ResponseWriter
	code  int
	wrote bool
}

func (w *defCodeWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.wrote = true
		w.WriteHeader(w.code)
	}
	return w.ResponseWriter.Write(p)
}

func (w *defCodeWriter) WriteHeader(statusCode int) {
	w.wrote = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func notFoundHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(&defCodeWriter{
			ResponseWriter: w,
			code:           http.StatusNotFound,
		}, r)
	}
}
