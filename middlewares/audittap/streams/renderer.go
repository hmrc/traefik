package streams

import (
	. "github.com/containous/traefik/middlewares/audittap/audittypes"
)

// Renderer is a function that encodes an audit summary.
type Renderer func(Summary) Encoded

//-------------------------------------------------------------------------------------------------

// DirectJSONRenderer is a Renderer that directly converts the summary to JSON.
func DirectJSONRenderer(summary Summary) Encoded {
	return summary.ToJson()
}

//-------------------------------------------------------------------------------------------------

type stream struct {
	renderer Renderer
	sink     AuditSink
}

func NewAuditStream(renderer Renderer, sink AuditSink) AuditStream {
	return &stream{renderer, sink}
}

func (s *stream) Audit(summary Summary) error {
	enc := s.renderer(summary)
	if enc.Err != nil {
		return enc.Err
	}
	return s.sink.Audit(enc)
}

func (s *stream) Close() error {
	return s.sink.Close()
}
