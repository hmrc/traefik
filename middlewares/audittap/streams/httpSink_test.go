package streams

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/containous/traefik/log"
	"github.com/stretchr/testify/assert"
)

func TestHttpSink(t *testing.T) {
	var got string

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		assert.NoError(t, err)
		got = string(body)
	}))
	defer stub.Close()

	w1, err := NewHTTPSink("PUT", stub.URL)
	assert.NoError(t, err)

	err = w1.Audit(encodedJSONSample)
	assert.NoError(t, err)

	assert.Equal(t, string(encodedJSONSample.Bytes), got)
}

type LogEvent struct {
	Level   string
	Message string
}

func TestLogEventsOnNon200Response(t *testing.T) {

	var buf bytes.Buffer
	log.SetOutput(&buf)

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer stub.Close()

	w1, _ := NewHTTPSink("POST", stub.URL)
	_ = w1.Audit(encodedJSONSample)
	logEventsStr := buf.String()
	logEvents := strings.Split(logEventsStr, "\n")
	lastLogEvent := logEvents[len(logEvents)-2]
	var logEvent LogEvent
	json.Unmarshal([]byte(lastLogEvent), &logEvent)
	assert.Equal(t, "warning", logEvent.Level)
	assert.Equal(t, "DS_EventMissed_AuditFailureResponse audit item : [1,2,3]", logEvent.Message)
}
