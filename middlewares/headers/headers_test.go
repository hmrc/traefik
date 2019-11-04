package headers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestHeaders_requestIdAdded(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	var requestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = r.Header.Get("X-Request-ID")
	})
	h := NewHeaders(true, "")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req, next)
	assert.True(t, strings.HasPrefix(requestID, "s"))
	id, _ := uuid.FromString(strings.TrimPrefix(requestID, "s"))
	assert.Equal(t, uint8(0x4), id.Version())
}

func TestHeaders_requestIdAddedNoS(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-ID-No-S", "true")
	var requestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = r.Header.Get("X-Request-ID")
	})
	h := NewHeaders(true, "")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req, next)
	assert.False(t, strings.HasPrefix(requestID, "s"))
	id, _ := uuid.FromString(requestID)
	assert.Equal(t, uint8(0x4), id.Version())
}

func TestHeaders_requestIdAlreadyPresent(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-ID", "itshouldbeme")
	var requestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = r.Header.Get("X-Request-ID")
	})
	h := NewHeaders(true, "")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req, next)
	assert.Equal(t, "itshouldbeme", requestID)
}

func TestHeaders_makesNewRequestIdForExistingHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-ID", "notme")
	var requestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = r.Header.Get("X-Request-ID")
	})
	h := NewHeaders(false, "")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req, next)
	assert.True(t, strings.HasPrefix(requestID, "s"))
	id, _ := uuid.FromString(strings.TrimPrefix(requestID, "s"))
	assert.Equal(t, uint8(0x4), id.Version())
}

func TestHeaders_requestIdIncludesLabel(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	var requestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = r.Header.Get("X-Request-ID")
	})
	h := NewHeaders(true, "my-label")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req, next)
	assert.True(t, strings.HasPrefix(requestID, "s-my-label-"))
	id, _ := uuid.FromString(strings.TrimPrefix(requestID, "s-my-label-"))
	assert.Equal(t, uint8(0x4), id.Version())
}
func TestHeaders_requestIdNoS_IncludesLabel(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-ID-No-S", "true")
	var requestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = r.Header.Get("X-Request-ID")
	})
	h := NewHeaders(true, "my-label")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req, next)
	assert.True(t, strings.HasPrefix(requestID, "my-label-"))
	id, _ := uuid.FromString(strings.TrimPrefix(requestID, "my-label-"))
	assert.Equal(t, uint8(0x4), id.Version())
}
