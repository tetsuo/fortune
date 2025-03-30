package dcensus

import (
	"net/http"
	"strings"

	"go.opencensus.io/plugin/ochttp"
)

// RouteTagger is a function type that takes an HTTP route and an incoming request,
// then returns a tagged string for monitoring purposes. It is used in OpenCensus
// instrumentation to label HTTP requests with custom route tags.
//
// Example: GET request to "/users/123" -> Route tag becomes "GET /users/:id"
//
//	customTagger := func(route string, r *http.Request) string {
//		normalizedRoute := strings.ReplaceAll(route, r.URL.Path, "/:id")
//		return fmt.Sprintf("%s %s", r.Method, normalizedRoute)
//	}
type RouteTagger func(route string, r *http.Request) string

// Router is an http multiplexer that instruments per-handler debugging
// information and census instrumentation.
type Router struct {
	http.Handler
	mux    *http.ServeMux
	tagger RouteTagger
}

// NewRouter creates a new Router, using tagger to tag incoming requests in monitoring.
// If tagger is nil, a default route tagger is used.
func NewRouter(tagger RouteTagger) *Router {
	if tagger == nil {
		tagger = func(route string, r *http.Request) string {
			return strings.Trim(route, "/")
		}
	}
	mux := http.NewServeMux()
	return &Router{
		mux:     mux,
		Handler: &ochttp.Handler{Handler: mux},
		tagger:  tagger,
	}
}

// Handle registers handler with the given route. It has the same routing semantics as http.ServeMux.
func (r *Router) Handle(route string, handler http.Handler) {
	r.mux.HandleFunc(route, func(w http.ResponseWriter, req *http.Request) {
		tag := r.tagger(route, req)
		ochttp.WithRouteTag(handler, tag).ServeHTTP(w, req)
	})
}

// HandleFunc is a wrapper around Handle for http.HandlerFuncs.
func (r *Router) HandleFunc(route string, handler http.HandlerFunc) {
	r.Handle(route, handler)
}
