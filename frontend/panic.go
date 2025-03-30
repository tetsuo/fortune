package frontend

import (
	"net/http"
)

// PanicHandler is invoked when some panic is caught by middleware.Panic.
func (s *Server) PanicHandler() (_ http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
