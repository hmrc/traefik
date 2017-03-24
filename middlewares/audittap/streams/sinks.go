package streams

import (
	"github.com/containous/traefik/middlewares/audittap/audittypes"
	"io"
)

//-------------------------------------------------------------------------------------------------

// AuditSink types handle encoded audit events.
type AuditSink interface {
	io.Closer
	// Audit handles an event
	Audit(encoded audittypes.Encoded) error
}

type noopAuditSink struct {
	audittypes.Encoded
}

var _ AuditSink = &noopAuditSink{} // prove type conformance

func (fs *noopAuditSink) Audit(encoded audittypes.Encoded) error {
	fs.Encoded = encoded
	return nil
}

func (fs *noopAuditSink) Close() error {
	return nil
}
