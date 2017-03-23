package audittap

import (
	. "github.com/containous/traefik/middlewares/audittap/audittypes"
	"github.com/containous/traefik/types"
	"net/http"
)

// MaximumEntityLength sets the upper limit for request and response entities. This will
// probably be removed in future versions.
const MaximumEntityLength = 32 * 1024

// AuditTap writes an event to the audit streams for every request.
type AuditTap struct {
	AuditStreams    []AuditStream
	Backend         string
	MaxEntityLength int
}

// NewAuditTap returns a new AuditTap handler.
func NewAuditTap(config *types.AuditTap, backend string) (*AuditTap, error) {
	//var renderer Renderer = DirectJSONRenderer

	//sinks, err := selectSinks(config, backend, renderer)
	//if err != nil {
	//	return nil, err
	//}

	var th int64 = MaximumEntityLength
	var err error
	if config.MaxEntityLength != "" {
		th, _, err = types.AsSI(config.MaxEntityLength)
		if err != nil {
			return nil, err
		}
	}

	return &AuditTap{nil, backend, int(th)}, nil
}

func (s *AuditTap) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	rhdr := NewHeaders(r.Header).DropHopByHopHeaders().SimplifyCookies().Flatten("hdr-")
	req := DataMap{
		Host:       r.Host,
		Method:     r.Method,
		Path:       r.URL.Path,
		Query:      r.URL.RawQuery,
		RemoteAddr: r.RemoteAddr,
		BeganAt:    TheClock.Now().UTC(),
	}
	req.AddAll(DataMap(rhdr))

	ww := NewAuditResponseWriter(rw, s.MaxEntityLength)
	next.ServeHTTP(ww, r)

	summary := Summary{s.Backend, req, ww.SummariseResponse()}
	for _, sink := range s.AuditStreams {
		sink.Audit(summary)
	}
}
