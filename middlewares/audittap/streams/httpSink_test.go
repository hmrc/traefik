package streams

import (
	"testing"
	"bytes"
	"github.com/containous/traefik/log"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
)

func TestHttpSink(t *testing.T) {
	var got string

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		assert.NoError(t, err)
		got = string(body)
	}))
	defer stub.Close()

	w1, err := NewHTTPSink("PUT", stub.URL, CreateClient())
	assert.NoError(t, err)

	err = w1.Audit(encodedJSONSample)
	assert.NoError(t, err)

	assert.Equal(t, string(encodedJSONSample.Bytes), got)
}

func TestHTTPConnectionIsReused(t *testing.T) {

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.Info("test do stuff with test server")

	// create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`OK`))
		//log.Info("Creating new http server")
	}))
	defer server.Close()

	// create a trace client
	clientTrace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { log.Printf("Conn was reused: %t", info.Reused) },
	}

	// request 1
	// construct a request
	req, err := http.NewRequest("GET", server.URL,nil)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), clientTrace))
	res, err := http.DefaultTransport.RoundTrip(req)

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close() // ensure the connection is closed regardless of the path taken
	assert.NoError(t, err)

	assert.True(t, strings.Contains(buf.String(), "Conn was reused: false"))
	assert.True(t, strings.Contains(buf.String(), "test do stuff with test server"))

	//  request 2
	res, err = http.DefaultTransport.RoundTrip(req)
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close() // ensure the connection is closed regardless of the path taken
	// request 3
	res, err = http.DefaultTransport.RoundTrip(req)
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close() // ensure the connection is closed regardless of the path taken
	assert.True(t, strings.Contains(buf.String(), "Conn was reused: true"))
	println(buf.String())

	ok(t, err)
}

