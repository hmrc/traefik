package streams

import (
	"bytes"
	"github.com/beeker1121/goque"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/middlewares/audittap/configuration"
	atypes "github.com/containous/traefik/middlewares/audittap/types"
	"github.com/stretchr/testify/assert"
	"os"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
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

/*
 * Do to the limitations of Go 1.12 there is no obvious way (to me) to be able to test the code in the way one
 * would like. The issue is that http and httptrace are different interfaces, so the clientTrace object can be
 * used within the NewHTTPSinkAsync as part of the config. So instead this test reflects the behaviour which is
 * applied in the code base to allow for http connections to be reused. New versions of Go (from 1.13) support a
 * method called NewRequestWithContext which would allow the context to be overriden in the tests.
 */
func TestHTTPAsyncConnectionIsReused(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`OK`))
	}))
	defer server.Close()

	client := CreateClient()

	// create a trace client
	clientTrace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { log.Printf("Conn was reused: %t", info.Reused) },
	}

	// request 1
	// construct a request
	req, err := http.NewRequest("GET", server.URL,nil)
	ok(t, err)
	reqCtx := req.WithContext(httptrace.WithClientTrace(req.Context(), clientTrace))

	sendRequest(client, encodedJSONSample, reqCtx)
	assert.True(t, strings.Contains(buf.String(), "Conn was reused: false"))

	// request 2
	req2, err2 := http.NewRequest("GET", server.URL,nil)
	ok(t, err2)
	reqCtx2 := req2.WithContext(httptrace.WithClientTrace(req2.Context(), clientTrace))
	sendRequest(client, encodedJSONSample, reqCtx2)

	assert.True(t, strings.Contains(buf.String(), "Conn was reused: true"))
}

func TestCreateClientWithoutCert(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	os.Setenv("CERTIFICATEPATH", "")
	CreateClient()
	assert.True(t, strings.Contains(buf.String(), "No CERTIFICATEPATH env var; reverting to http client"))
}

func TestCreateClientWithCert(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	os.Setenv("CERTIFICATEPATH", "/go/src/github.com/containous/traefik/examples/test.key")
	CreateClient()

	assert.True(t, strings.Contains(buf.String(), "Cert:[45 45 45 45 45 66 69 71 73 78 32 80 82 73 86 65 84 69 32 75]"))
}