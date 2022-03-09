package streams

import (
	"bytes"
	"github.com/beeker1121/goque"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/middlewares/audittap/configuration"
	atypes "github.com/containous/traefik/middlewares/audittap/types"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogEventsOnNon200Response(t *testing.T) {

	var buf bytes.Buffer
	log.SetOutput(&buf)

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer stub.Close()

	var config = configuration.AuditSink{
		Endpoint:      stub.URL,
		Destination:   "bar",
		DiskStorePath: "/tmp/test",
		NumProducers:  1,
	}

	q, err := goque.OpenQueue(config.DiskStorePath)

	if err != nil {
		t.Fatal("failed to open queue", err)
	}

	NewQueue = func(queueLocation string) (*goque.Queue, error) {
		return q, err
	}

	messages := make(chan atypes.Encoded, 1)
	w1, _ := NewHTTPSinkAsync(&config, messages)
	_ = w1.Audit(encodedJSONSample)
	time.Sleep(1000 * time.Millisecond)
	assert.True(t, strings.Contains(buf.String(), `{"level":"warning","message":"DS_EventMissed_AuditFailureResponse audit item : [1,2,3]"`))
}

func TestHttpClientIsAsync(t *testing.T) {

	var buf bytes.Buffer
	log.SetOutput(&buf)

	var sleepTime time.Duration
	sleepTime = 2000 * time.Millisecond

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(sleepTime)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer stub.Close()

	var config = configuration.AuditSink{
		Endpoint:      stub.URL,
		Destination:   "bar",
		DiskStorePath: "/tmp/test",
		NumProducers:  1,
	}
	messages := make(chan atypes.Encoded, 1)
	w1, _ := NewHTTPSinkAsync(&config, messages)
	t1 := time.Now()
	_ = w1.Audit(encodedJSONSample)
	t2 := time.Now()
	timeItTook := t2.Sub(t1)
	assert.True(t, timeItTook < sleepTime, "The program should complete quicker than 'sleepTime'")
}
