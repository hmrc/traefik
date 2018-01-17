package audittypes

import (
	"net/http"

	"github.com/containous/traefik/middlewares/audittap/types"
)

// APIAuditEvent is the audit event created for API calls
type APIAuditEvent struct {
	AuditEvent
	AuthorisationToken string `json:"authorisationToken,omitempty"`
}

// AppendRequest appends information about the request to the audit event
func (ev *APIAuditEvent) AppendRequest(req *http.Request) {
	reqHeaders := appendCommonRequestFields(&ev.AuditEvent, req)
	ev.AuthorisationToken = reqHeaders.GetString("authorization")
}

// AppendResponse appends information about the response to the audit event
func (ev *APIAuditEvent) AppendResponse(responseHeaders http.Header, respInfo types.ResponseInfo) {
	appendCommonResponseFields(&ev.AuditEvent, responseHeaders, respInfo)
}

// EnforceConstraints ensures the audit event satisfies constraints
func (ev *APIAuditEvent) EnforceConstraints(constraints AuditConstraints) bool {
	enforcePrecedentConstraints(&ev.AuditEvent, constraints)
	return true
}

// ToEncoded transforms the event into an Encoded
func (ev *APIAuditEvent) ToEncoded() types.Encoded {
	return types.ToEncoded(ev)
}

// NewAPIAuditEvent creates a new APIAuditEvent with the provided auditSource and auditType
func NewAPIAuditEvent(auditSource string, auditType string) Auditer {
	ev := APIAuditEvent{}
	ev.AuditEvent = AuditEvent{AuditSource: auditSource, AuditType: auditType}
	return &ev
}
