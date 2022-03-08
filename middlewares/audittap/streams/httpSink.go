package streams

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/containous/traefik/middlewares/audittap/types"
	log "github.com/sirupsen/logrus"
)

type httpSink struct {
	method, endpoint string
	client *http.Client
}

// NewHTTPSink creates a new HTTP sink
func NewHTTPSink(method, endpoint string, client *http.Client) (AuditSink, error) {
	if method == "" {
		method = http.MethodPost
	}
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Cannot access endpoint '%s': %v", endpoint, err)
	}

	return &httpSink{method, endpoint, client}, nil
}

func CreateClient() (*http.Client) {
	var certPath = os.Getenv("CERTIFICATEPATH")
	var client *http.Client

	if len(certPath) > 0 {
		caCert, err := ioutil.ReadFile(certPath)
		if err != nil {
			log.Info("Error Cert Read ", err)
		} else {
			log.Info("Cert:", caCert[0:20])
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
	} else {
		log.Warn("No CERTIFICATEPATH env var; reverting to http client")
		client = &http.Client {} // no certificate specified or cert not found
	}

	return client
}

func (has *httpSink) Audit(encoded types.Encoded) error {

	request, err := http.NewRequest(has.method, has.endpoint, bytes.NewBuffer(encoded.Bytes))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Length", fmt.Sprintf("%d", encoded.Length()))

	res, err := has.client.Do(request)

	if err != nil || res.StatusCode > 299 {
		log.SetFormatter(&log.JSONFormatter{
			FieldMap: log.FieldMap{
				log.FieldKeyMsg: "message",
			},
		})
		log.Warn("DS_EventMissed_AuditFailureResponse audit item : " + string(encoded.Bytes))
		return err
	}
	// close the http body before making a new http request: https://golang.cafe/blog/how-to-reuse-http-connections-in-go.html
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close() // ensure the connection is closed regardless of the path taken
	return nil
}

func (has *httpSink) Close() error {
	return nil
}
