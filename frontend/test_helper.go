package frontend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tetsuo/fortune/internal/database"
	"github.com/tetsuo/fortune/internal/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestServer(t *testing.T, testDB *database.DB) (*Server, http.Handler, *observer.ObservedLogs) {
	t.Helper()

	s, err := NewServer(
		Config{},
		testDB,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()

	s.Install(mux.Handle)

	middlewareStack := middleware.Chain(
		middleware.AcceptRequests(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodHead),
		middleware.Timeout(54*time.Second),
	)

	zc, o := observer.New(zap.DebugLevel)
	s.log = zap.New(zc).Sugar()

	return s, middlewareStack(mux), o
}

type wantedLog struct {
	severity string
	message  string
}

type ttest struct {
	name        string
	path        string
	contentType string
	method      string
	headers     map[string]string
	body        []byte
	wantStatus  int
	wantText    string
	wantEmpty   bool
	wantLogs    []wantedLog
	wantHeaders map[string][]string
}

func (tt ttest) run(t *testing.T, handler http.Handler, observedLogs *observer.ObservedLogs) {
	t.Helper()

	w := httptest.NewRecorder()
	r := tt.getRequest(t)
	r.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")

	handler.ServeHTTP(w, r)

	res := w.Result()
	defer func() {
		_ = res.Body.Close()
	}()

	assert.Equal(t, tt.wantStatus, res.StatusCode)

	if tt.wantHeaders != nil {
		for key := range tt.wantHeaders {
			assert.Equal(t, tt.wantHeaders[key], res.Header.Values(key))
		}
	}

	if tt.wantText != "" {
		assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-type"))
		assert.Equal(t, tt.wantText, w.Body.String())
	} else if tt.wantEmpty {
		assert.Equal(t, w.Body.Len(), 0)
	}

	logz := observedLogs.TakeAll()
	wantLogsLen := len(tt.wantLogs)
	if wantLogsLen > 0 {
		assert.Len(t, logz, wantLogsLen, fmt.Sprintf("expected %d log lines, got %d", wantLogsLen, len(logz)))
		var message string
		for i, loggedEntry := range logz {
			// Fix spaces in log lines acting up due to a issue in Printf(%q)
			message = strings.Join(strings.Fields(loggedEntry.Message), " ")

			assert.Equal(t, tt.wantLogs[i].message, message)
			assert.Equal(t, strings.ToUpper(tt.wantLogs[i].severity), loggedEntry.Level.CapitalString())
		}
	} else {
		assert.Empty(t, logz)
	}
}

func (tt ttest) getRequest(t *testing.T) *http.Request {
	t.Helper()

	method := tt.method
	if method == "" {
		method = "GET"
	}
	body := tt.body
	path := tt.path
	if path == "" {
		path = "/"
	}

	var r *http.Request
	var br io.Reader

	if body != nil {
		br = bytes.NewReader(tt.body)
		r = httptest.NewRequest(method, path, br)
	} else {
		r = httptest.NewRequest(method, path, nil)
	}

	if tt.contentType != "" {
		r.Header.Set("Content-Type", tt.contentType)
	}

	for k, v := range tt.headers {
		r.Header.Set(k, v)
	}

	return r
}
