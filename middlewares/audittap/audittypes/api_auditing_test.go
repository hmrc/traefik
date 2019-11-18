package audittypes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"encoding/json"

	"github.com/containous/traefik/middlewares/audittap/types"
	"github.com/stretchr/testify/assert"
)

func TestApiAuditEvent(t *testing.T) {

	requestBody, _ := json.Marshal(types.DataMap{
		"foo": "bar",
		"baz": "biff",
	})

	responseBody, _ := json.Marshal(types.DataMap{
		"respFoo": "respBar",
		"respBaz": "respBiff",
	})

	ev := APIAuditEvent{}
	req := httptest.NewRequest("POST", "/some/api/resource?p1=v1", bytes.NewReader(requestBody))
	req.Header.Set("Authorization", "auth456")

	respHdrs := http.Header{}
	respHdrs.Set("Content-Type", "text/plain")
	respInfo := types.ResponseInfo{404, 101, responseBody, 2048}

	spec := &AuditSpecification{}
	ev.AppendRequest(NewRequestContext(req), spec)
	ev.AppendResponse(respHdrs, respInfo, spec)

	assert.Equal(t, "POST", ev.Method)
	assert.Equal(t, "/some/api/resource", ev.Path)
	assert.Equal(t, "", ev.ProxiedPath)
	assert.Equal(t, "p1=v1", ev.QueryString)
	assert.Equal(t, "auth456", ev.AuthorisationToken)

	assert.EqualValues(t, len(requestBody), ev.RequestPayload.Get("length"))
	assert.Equal(t, string(requestBody), ev.RequestPayload["contents"])

	assert.EqualValues(t, len(responseBody), ev.ResponsePayload.Get("length"))
	assert.Equal(t, string(responseBody), ev.ResponsePayload["contents"])

	assert.Equal(t, "404", ev.ResponseStatus)

	assert.True(t, ev.EnforceConstraints(AuditConstraints{}))
}

func TestApiGatewayPrefixRulePathChange(t *testing.T) {

	ev := APIAuditEvent{}
	req := httptest.NewRequest("POST", "/current/api/resource?p1=v1", nil)
	req.Header.Set("X-Forwarded-Prefix", "/the/actual/service/")

	spec := &AuditSpecification{}
	ev.AppendRequest(NewRequestContext(req), spec)

	assert.Equal(t, "/the/actual/service/api/resource", ev.Path)
	assert.Equal(t, "/current/api/resource", ev.ProxiedPath)
	assert.Equal(t, "p1=v1", ev.QueryString)
}

func TestDefinePathReplacementOnlyIsForPrefix(t *testing.T) {

	ev := APIAuditEvent{}
	req := httptest.NewRequest("POST", "/future/current/api/resource?p1=v1", nil)
	req.Header.Set("X-Forwarded-Prefix", "/the/actual/service/")

	spec := &AuditSpecification{}
	ev.AppendRequest(NewRequestContext(req), spec)

	assert.Equal(t, "/future/current/api/resource", ev.Path)
	assert.Equal(t, "", ev.ProxiedPath)
	assert.Equal(t, "p1=v1", ev.QueryString)
}

func TestFormEncodedContentMasking(t *testing.T) {

	requestBody := "say=Hi&password=ishouldbesecret&secret=notforyoureyes&to=Dave"

	ev := APIAuditEvent{}
	req := httptest.NewRequest("POST", "/some/api/resource?p1=v1", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=ISO-8859-1")

	obfuscate := AuditObfuscation{MaskFields: []string{"password", "secret"}, MaskValue: "@@@"}
	spec := &AuditSpecification{}
	spec.AuditObfuscation = obfuscate
	ev.AppendRequest(NewRequestContext(req), spec)

	expectedBody := "say=Hi&password=@@@&secret=@@@&to=Dave"
	assert.EqualValues(t, len(requestBody), ev.RequestPayload.Get("length"))
	assert.Equal(t, expectedBody, ev.RequestPayload["contents"])

	assert.True(t, ev.EnforceConstraints(AuditConstraints{}))
}

func TestJsonContentMasking(t *testing.T) {

	requestBody := `{
		"password": "keepmesecret",
		"foo": "bar",
		"secret": "notforyoureyes",
		"baz": "phew"
	}`

	ev := APIAuditEvent{}
	req := httptest.NewRequest("POST", "/some/api/resource?p1=v1", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	obfuscate := AuditObfuscation{MaskFields: []string{"password", "secret"}, MaskValue: "@@@"}
	spec := &AuditSpecification{}
	spec.AuditObfuscation = obfuscate
	ev.AppendRequest(NewRequestContext(req), spec)
	expectedBody := `{
		"password": "@@@",
		"foo": "bar",
		"secret": "@@@",
		"baz": "phew"
	}`

	assert.EqualValues(t, len(requestBody), ev.RequestPayload.Get("length"))
	assert.Equal(t, expectedBody, ev.RequestPayload["contents"])

	assert.True(t, ev.EnforceConstraints(AuditConstraints{}))
}

func TestNewApiAudit(t *testing.T) {
	auditer := NewAPIAuditEvent("ping", "pong")
	if api, ok := auditer.(*APIAuditEvent); ok {
		assert.Equal(t, "ping", api.AuditSource)
		assert.Equal(t, "pong", api.AuditType)
	} else {
		assert.Fail(t, "Was not an APIAuditEvent")
	}
}
