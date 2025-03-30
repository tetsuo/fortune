package frontend

import (
	"crypto/rand"
	"net/http"
	"testing"
)

func TestPOST(t *testing.T) {
	t.Parallel()

	testDB, release := acquire(t)
	defer release()

	_, handler, observedLogs := newTestServer(t, testDB)

	for _, tt := range []ttest{
		{
			name:        "invalid content type",
			method:      "POST",
			contentType: "application/json",
			path:        "/",
			body:        []byte(`{"some": "json"}`),
			wantStatus:  http.StatusUnsupportedMediaType,
			wantText:    "Unsupported Media Type\n",
			wantLogs: []wantedLog{
				{
					"info",
					`415 <nil>`,
				},
			},
		},
		{
			name:        "request entity too large",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			body: func() []byte {
				bigdata := make([]byte, (1<<20)+1) // it is 1 byte more than 1MB

				_, err := rand.Read(bigdata)
				if err != nil {
					panic(err)
				}

				return bigdata
			}(),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantText:   "Request Entity Too Large\n",
			wantLogs: []wantedLog{
				{
					"info",
					`413 http: request body too large`,
				},
			},
		},
		{
			name:        "empty request",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			wantStatus:  http.StatusBadRequest,
			wantText:    "Bad Request\n",
			wantLogs: []wantedLog{
				{
					"info",
					`400 <nil>`,
				},
			},
		},
		{
			name:        "invalid fortune 1",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			body:        []byte(`%`),
			wantStatus:  http.StatusBadRequest,
			wantText:    "Bad Request\n",
			wantLogs: []wantedLog{
				{
					"info",
					`400 <nil>`,
				},
			},
		},
		{
			name:        "invalid fortune 2",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			body:        []byte("%\n%\n%\n"),
			wantStatus:  http.StatusBadRequest,
			wantText:    "Bad Request\n",
			wantLogs: []wantedLog{
				{
					"info",
					`400 <nil>`,
				},
			},
		},
		{
			name:        "single fortune",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			body:        []byte("hoi"),
			wantStatus:  http.StatusCreated,
			wantHeaders: map[string][]string{
				"X-Inserted-Count": {"1"},
			},
			wantLogs: []wantedLog{},
		},
		{
			name:        "multiple fortunes",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			body:        []byte("mul\n%\ntiple\n\n\n%\ncppkies"),
			wantStatus:  http.StatusCreated,
			wantHeaders: map[string][]string{
				"X-Inserted-Count": {"3"},
			},
			wantLogs: []wantedLog{},
		},
		{
			name:        "batch with empty fortunes",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			body:        []byte("bar\n%\n%baz\n%\n    \n%\n\n\n\n%\n"),
			wantStatus:  http.StatusCreated,
			wantHeaders: map[string][]string{
				"X-Inserted-Count": {"2"},
			},
			wantLogs: []wantedLog{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, handler, observedLogs)
		})
	}
}

func TestGET(t *testing.T) {
	t.Parallel()

	testDB, release := acquire(t)
	defer release()

	_, handler, observedLogs := newTestServer(t, testDB)

	expectedFortune := `"bazbar
ququux"
		―  Romantic Hacker`

	for _, tt := range []ttest{
		{
			name:       "no rows in the resultset",
			method:     "GET",
			path:       "/",
			wantStatus: http.StatusNotFound,
			wantText:   "Not Found\n",
			wantLogs: []wantedLog{
				{
					"info",
					`404 sql: no rows in result set`,
				},
			},
		},
		{
			name:        "insert a fortune to get",
			method:      "POST",
			contentType: "text/plain",
			path:        "/",
			body:        []byte(expectedFortune),
			wantStatus:  http.StatusCreated,
			wantHeaders: map[string][]string{
				"X-Inserted-Count": {"1"},
			},
		},
		{
			name:       "get a fortune",
			method:     "GET",
			path:       "/",
			wantStatus: http.StatusOK,
			wantText:   expectedFortune,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, handler, observedLogs)
		})
	}
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	_, handler, observedLogs := newTestServer(t, nil)

	for _, tt := range []ttest{
		{
			name:       "invalid GET path",
			method:     "GET",
			path:       "/notmuch",
			wantStatus: http.StatusNotFound,
			wantText:   "Not Found\n",
		},
		{
			name:       "invalid POST path",
			method:     "POST",
			path:       "/notmuch",
			wantStatus: http.StatusNotFound,
			wantText:   "Not Found\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, handler, observedLogs)
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()

	_, handler, observedLogs := newTestServer(t, nil)

	for _, tt := range []ttest{
		{
			name:       "method not allowed",
			method:     "PUT",
			path:       "/",
			wantStatus: http.StatusMethodNotAllowed,
			wantText:   "Method Not Allowed\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, handler, observedLogs)
		})
	}
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	_, handler, observedLogs := newTestServer(t, nil)

	for _, tt := range []ttest{
		{
			name:       "healthz - ok",
			method:     "GET",
			path:       "/healthz",
			wantStatus: http.StatusOK,
			wantEmpty:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, handler, observedLogs)
		})
	}
}
