package middlewares

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/containous/traefik/middlewares/accesslog"
)

var (
	_ Stateful = &responseRecorder{}
)

// StatsRecorder is an optional middleware that records more details statistics
// about requests and how they are processed. This currently consists of recent
// requests that have caused errors (4xx and 5xx status codes), making it easy
// to pinpoint problems.
type StatsRecorder struct {
	Stats
	mutex           sync.RWMutex
	backendReqMutex sync.RWMutex
	numRecentErrors int
	connStateMutex  sync.RWMutex
	connStateMap    map[net.Conn]http.ConnState
}

// NewStatsRecorder returns a new StatsRecorder
func NewStatsRecorder(numRecentErrors int) *StatsRecorder {
	return &StatsRecorder{
		Stats: Stats{
			BackendRequests: make(map[string]int64),
		},
		numRecentErrors: numRecentErrors,
		connStateMap:    make(map[net.Conn]http.ConnState),
	}
}

// Stats includes all of the stats gathered by the recorder.
type Stats struct {
	RecentErrors       []*statsError    `json:"recent_errors"`
	ActiveConnections  int64            `json:"active_connections"`
	IdleConnections    int64            `json:"idle_connections"`
	CurrentConnections int64            `json:"current_connections"`
	HandledConnections int64            `json:"handled_connections"`
	BackendRequests    map[string]int64 `json:"backend_requests"`
}

// statsError represents an error that has occurred during request processing.
type statsError struct {
	StatusCode int       `json:"status_code"`
	Status     string    `json:"status"`
	Method     string    `json:"method"`
	Host       string    `json:"host"`
	Path       string    `json:"path"`
	Time       time.Time `json:"time"`
}

// responseRecorder captures information from the response and preserves it for
// later analysis.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code for later retrieval.
func (r *responseRecorder) WriteHeader(status int) {
	r.ResponseWriter.WriteHeader(status)
	r.statusCode = status
}

// Hijack hijacks the connection
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify returns a channel that receives at most a
// single value (true) when the client connection has gone
// away.
func (r *responseRecorder) CloseNotify() <-chan bool {
	return r.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Flush sends any buffered data to the client.
func (r *responseRecorder) Flush() {
	r.ResponseWriter.(http.Flusher).Flush()
}

// ServeHTTP silently extracts information from the request and response as it
// is processed. If the response is 4xx or 5xx, add it to the list of 10 most
// recent errors.
func (s *StatsRecorder) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	recorder := &responseRecorder{w, http.StatusOK}
	next(recorder, r)
	if recorder.statusCode >= 400 {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.RecentErrors = append([]*statsError{
			{
				StatusCode: recorder.statusCode,
				Status:     http.StatusText(recorder.statusCode),
				Method:     r.Method,
				Host:       r.Host,
				Path:       r.URL.Path,
				Time:       time.Now(),
			},
		}, s.RecentErrors...)
		// Limit the size of the list to numRecentErrors
		if len(s.RecentErrors) > s.numRecentErrors {
			s.RecentErrors = s.RecentErrors[:s.numRecentErrors]
		}
	}

	s.incTotalRequestsForBackend(r)
}

// Data returns a copy of the statistics that have been gathered.
func (s *StatsRecorder) Data() *Stats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// We can't return the slice directly or a race condition might develop
	recentErrors := make([]*statsError, len(s.RecentErrors))
	copy(recentErrors, s.RecentErrors)

	res := s.Stats
	res.RecentErrors = recentErrors
	return &res
}

// ConnStateChange updates stats on change of connection state.
func (s *StatsRecorder) ConnStateChange(conn net.Conn, newState http.ConnState) {
	s.connStateMutex.Lock()
	defer s.connStateMutex.Unlock()

	oldState := s.connStateMap[conn]

	switch newState {
	case http.StateActive:
		switch oldState {
		case http.StateIdle:
			atomic.AddInt64(&s.IdleConnections, -1)
		}
		atomic.AddInt64(&s.ActiveConnections, 1)

	case http.StateIdle:
		switch oldState {
		case http.StateActive:
			atomic.AddInt64(&s.ActiveConnections, -1)
		}
		atomic.AddInt64(&s.IdleConnections, 1)

	case http.StateNew:
		s.connStateMap[conn] = newState
		atomic.AddInt64(&s.CurrentConnections, 1)
		atomic.AddInt64(&s.HandledConnections, 1)

	case http.StateHijacked:
		fallthrough

	case http.StateClosed:
		switch oldState {
		case http.StateActive:
			atomic.AddInt64(&s.ActiveConnections, -1)
		case http.StateIdle:
			atomic.AddInt64(&s.IdleConnections, -1)
		}
		atomic.AddInt64(&s.CurrentConnections, -1)

		delete(s.connStateMap, conn)
		return
	}

	s.connStateMap[conn] = newState
}

func (s *StatsRecorder) incTotalRequestsForBackend(r *http.Request) {
	s.backendReqMutex.Lock()
	defer s.backendReqMutex.Unlock()

	var backendName, backendAddr interface{}
	var ok bool

	logTable := r.Context().Value(accesslog.DataTableKey)

	if logTable == nil {
		return
	}

	if backendName, ok = logTable.(*accesslog.LogData).Core[accesslog.BackendName]; !ok {
		return
	}

	if backendAddr, ok = logTable.(*accesslog.LogData).Core[accesslog.BackendAddr]; !ok {
		return
	}

	statName := fmt.Sprintf("%s_%s", backendName, backendAddr.(string))

	s.BackendRequests[statName]++
}
