package streams

import (
	. "github.com/containous/traefik/middlewares/audittap/audittypes"
	"io"
)

//-------------------------------------------------------------------------------------------------

type AuditSink interface {
	io.Closer
	Audit(encoded Encoded) error
}

type noopAuditSink struct {
	Encoded
}

var _ AuditSink = &noopAuditSink{} // prove type conformance

func (fs *noopAuditSink) Audit(encoded Encoded) error {
	fs.Encoded = encoded
	return nil
}

func (fs *noopAuditSink) Close() error {
	return nil
}
