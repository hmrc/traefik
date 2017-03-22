package audittap

import (
	"fmt"
	"github.com/containous/traefik/types"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fixedClock time.Time

func (c fixedClock) Now() time.Time {
	return time.Time(c)
}

var t0 = fixedClock(time.Unix(1000000000, 0))

type noopAuditStream struct {
	events []Summary
}

func (as *noopAuditStream) Audit(summary Summary) error {
	as.events = append(as.events, summary)
	return nil
}

func TestAuditTap_noop(t *testing.T) {
	clock = t0

	capture := &noopAuditStream{}
	cfg := &types.AuditTap{}
	tap, err := NewAuditTap(cfg, "backend1")
	tap.AuditStreams = []AuditStream{capture}
	assert.NoError(t, err)

	req := httptest.NewRequest("", "/a/b/c?d=1&e=2", nil)
	req.RemoteAddr = "101.102.103.104:1234"
	req.Host = "example.co.uk"
	req.Header.Set("Request-ID", "R123")
	req.Header.Set("Session-ID", "S123")
	res := httptest.NewRecorder()

	tap.ServeHTTP(res, req, http.HandlerFunc(notFound))

	assert.Equal(t, 1, len(capture.events))
	assert.Equal(t,
		Summary{
			"backend1",
			DataMap{
				Host:             "example.co.uk",
				Method:           "GET",
				Path:             "/a/b/c",
				Query:            "d=1&e=2",
				RemoteAddr:       "101.102.103.104:1234",
				"hdr-request-id": "R123",
				"hdr-session-id": "S123",
				BeganAt:          clock.Now(),
			},
			DataMap{
				Status: 404,
				"hdr-x-content-type-options": "nosniff",
				"hdr-content-type":           "text/plain; charset=utf-8",
				Size:                         19,
				Entity:                       []byte("404 page not found\n"),
				CompletedAt:                  clock.Now(),
			},
		},
		capture.events[0])
}

// simpleHandler replies to the request with the specified error message and HTTP code.
// It does not otherwise end the request; the caller should ensure no further
// writes are done to w.
// The error message should be plain text.
func simpleHandler(w http.ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, error)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	simpleHandler(w, "404 page not found", http.StatusNotFound)
}
