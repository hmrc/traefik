package audittap

import (
	"github.com/containous/traefik/middlewares/audittap/audittypes"
	"github.com/containous/traefik/types"
	"net/http"
)

// MaximumEntityLength sets the upper limit for request and response entities. This will
// probably be removed in future versions.
const MaximumEntityLength = 32 * 1024

// AuditTap writes an event to the audit streams for every request.
type AuditTap struct {
	AuditStreams    []audittypes.AuditStream
	Backend         string
	MaxEntityLength int
	next            http.Handler
}

// NewAuditTap returns a new AuditTap handler.
func NewAuditTap(config *types.AuditSink, streams []audittypes.AuditStream, backend string, next http.Handler) (*AuditTap, error) {
	var th int64 = MaximumEntityLength
	var err error
	if config.MaxEntityLength != "" {
		th, _, err = types.AsSI(config.MaxEntityLength)
		if err != nil {
			return nil, err
		}
	}

	return &AuditTap{streams, backend, int(th), next}, nil
}

func (s *AuditTap) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rhdr := NewHeaders(r.Header).DropHopByHopHeaders().SimplifyCookies().Flatten("hdr-")
	req := audittypes.DataMap{
		audittypes.Host:       r.Host,
		audittypes.Method:     r.Method,
		audittypes.Path:       r.URL.Path,
		audittypes.Query:      r.URL.RawQuery,
		audittypes.RemoteAddr: r.RemoteAddr,
		audittypes.BeganAt:    audittypes.TheClock.Now().UTC(),
	}
	req.AddAll(audittypes.DataMap(rhdr))

	ww := NewAuditResponseWriter(rw, s.MaxEntityLength)
	s.next.ServeHTTP(ww, r)

	summary := audittypes.Summary{s.Backend, req, ww.SummariseResponse()}
	for _, sink := range s.AuditStreams {
		sink.Audit(summary)
	}
}
