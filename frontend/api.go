package frontend

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// servePOST handles HTTP POST requests to insert new fortune messages.
// It validates the request content type, enforces a maximum body size,
// and parses the input using decodeBody. The parsed values are then
// inserted into the database in bulk. Returns an error if validation,
// parsing, or database insertion fails.
func (s *Server) servePOST(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil
	}

	ct := r.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || mediaType != "text/plain" {
		return &serverError{
			status:       http.StatusUnsupportedMediaType,
			responseText: http.StatusText(http.StatusUnsupportedMediaType),
			err:          err,
		}
	}

	const maxBodySize = 1 << 20 // 1 MB in bytes

	values, err := decodeBody(http.MaxBytesReader(w, r.Body, maxBodySize))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return &serverError{
				status:       http.StatusRequestEntityTooLarge,
				responseText: http.StatusText(http.StatusRequestEntityTooLarge),
				err:          maxErr,
			}
		}
		// Other errors
		return err
	}

	insertCount := len(values)

	if insertCount < 1 {
		return &serverError{
			status:       http.StatusBadRequest,
			responseText: http.StatusText(http.StatusBadRequest),
		}
	}

	w.Header().Set("X-Inserted-Count", strconv.Itoa(insertCount))

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err = s.db.BulkInsert(ctx, "fortune_cookies", []string{"value"}, values, ""); err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)

	return nil
}

// serveGET handles HTTP GET requests to retrieve a random fortune message.
// It selects a random entry from the database and returns it as a plain
// text response. Returns an error if querying the database fails.
func (s *Server) serveGET(w http.ResponseWriter, r *http.Request) error {
	if r.URL.Path != "/" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var message []byte

	err := s.db.QueryRow(ctx, `SELECT value
FROM fortune_cookies
WHERE id >= (
   SELECT FLOOR( RAND() * (SELECT MAX(id) FROM fortune_cookies) ) + 1
)
ORDER BY id
LIMIT 1`).Scan(&message)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &serverError{
				status:       http.StatusNotFound,
				responseText: http.StatusText(http.StatusNotFound),
				err:          err,
			}
		}
		// Other errors
		return err
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	_, err = w.Write(message)

	return err
}

// decodeBody parses the fortune format from a request body, splitting messages by '%'
// and trimming whitespace. It filters out invalid lengths and returns a slice of any
// since BulkInsert requires it. Returns an error if reading fails.
func decodeBody(r io.Reader) ([]any, error) {
	const (
		minCookieLength = 3
		maxCookieLength = 10000
	)

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	input := string(data)
	var cookies []any

	// Split input into lines first
	lines := strings.Split(input, "\n")

	var currentBlock []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// If a line contains only '%', it means we reached a separator.
		if trimmed == "%" {
			if len(currentBlock) > 0 {
				cookie := strings.TrimSpace(strings.Join(currentBlock, "\n"))

				length := len(cookie)
				if length >= minCookieLength && length <= maxCookieLength {
					cookies = append(cookies, any(cookie))
				}

				currentBlock = nil
			}
			continue
		}

		currentBlock = append(currentBlock, line)
	}

	// Handle last block (if there's no trailing '%')
	if len(currentBlock) > 0 {
		cookie := strings.TrimSpace(strings.Join(currentBlock, "\n"))
		length := len(cookie)
		if length >= minCookieLength && length <= maxCookieLength {
			cookies = append(cookies, any(cookie))
		}
	}

	return cookies, nil
}
