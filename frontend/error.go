//lint:file-ignore U1000 tbd
package frontend

import (
	"errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/errorreporting"
	"github.com/tetsuo/fortune/internal/middleware"
	"github.com/tetsuo/fortune/internal/wraperr"
)

// serverError represents a structured error with an HTTP status code,
// a user-facing response message, and an underlying wrapped error.
type serverError struct {
	status       int    // HTTP status code
	responseText string // Response text to the user
	err          error  // wrapped error
}

// Error implements the error interface for serverError, returning a formatted string
// containing the HTTP status code and the underlying error message.
func (s *serverError) Error() string {
	return fmt.Sprintf("%d %v", s.status, s.err)
}

// Unwrap returns the underlying error wrapped by serverError to allow for
// error unwrapping and further processing.
func (s *serverError) Unwrap() error {
	return s.err
}

// serveError handles errors returned from request handlers by determining the appropriate
// HTTP response status, logging the error, and reporting it to an error tracking service
// if applicable.
func (s *Server) serveError(w http.ResponseWriter, r *http.Request, err error) {
	var serr *serverError
	if !errors.As(err, &serr) {
		serr = &serverError{status: http.StatusInternalServerError, err: err}
	}
	if serr.status == http.StatusInternalServerError {
		s.log.Error(err)
		s.reportError(err, w, r)
	} else {
		s.log.Info(err)
	}
	if serr.responseText == "" {
		serr.responseText = http.StatusText(serr.status)
	}
	http.Error(w, serr.responseText, serr.status)
}

// errorHandler is a middleware wrapper for HTTP handlers that automatically
// captures errors returned by handlers and passes them to serveError for
// proper handling and logging.
func (s *Server) errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			s.serveError(w, r, err)
		}
	}
}

// reportError sends captured errors to Google Cloud Error Reporting,
// including any extracted stack trace if available.
func (s *Server) reportError(err error, w http.ResponseWriter, r *http.Request) {
	if s.er == nil {
		return
	}
	// Extract the stack trace from the error if there is one.
	var stack []byte
	if serr := (*wraperr.StackError)(nil); errors.As(err, &serr) {
		stack = serr.Stack
	}
	if s.er != nil {
		s.er.Report(errorreporting.Entry{
			Error: err,
			Req:   r,
			Stack: stack,
		})
	}
	s.log.Debugf("reported error %v with stack size %d", err, len(stack))
	// Bypass the error-reporting middleware.
	w.Header().Set(middleware.BypassErrorReportingHeader, "true")
}
