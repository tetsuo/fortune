// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"fmt"
	"net/http"

	"cloud.google.com/go/errorreporting"
)

// BypassErrorReportingHeader is the header key used by the ErrorReporting middleware
// to avoid calling the errorreporting service.
const BypassErrorReportingHeader = "X-Fortune-Bypass-Error-Reporting"

// ErrorReporting returns a middleware that reports any server errors using the
// report func.
func ErrorReporting(report func(errorreporting.Entry)) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w2 := &erResponseWriter{ResponseWriter: w}
			h.ServeHTTP(w2, r)
			// Don't report if the bypass header was set.
			if w2.bypass {
				return
			}
			// Don't report success or client errors.
			if w2.status < 500 {
				return
			}

			report(errorreporting.Entry{
				Error: fmt.Errorf("handler for %q returned status code %d", r.URL.Path, w2.status),
				Req:   r,
			})
		})
	}
}

type erResponseWriter struct {
	http.ResponseWriter

	bypass bool
	status int
}

func (rw *erResponseWriter) WriteHeader(code int) {
	rw.status = code
	if rw.ResponseWriter.Header().Get(BypassErrorReportingHeader) == "true" {
		rw.bypass = true
		// Don't send this header to clients.
		rw.ResponseWriter.Header().Del(BypassErrorReportingHeader)
	}
	rw.ResponseWriter.WriteHeader(code)
}
