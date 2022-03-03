package streams

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/containous/traefik/middlewares/audittap/types"
	log "github.com/sirupsen/logrus"
)

type httpSink struct {
	method, endpoint string
}

// NewHTTPSink creates a new HTTP sink
func NewHTTPSink(method, endpoint string) (AuditSink, error) {
	if method == "" {
		method = http.MethodPost
	}
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Cannot access endpoint '%s': %v", endpoint, err)
	}
	return &httpSink{method, endpoint}, nil
}

func (has *httpSink) Audit(encoded types.Encoded) error {

	log.Info("Audit HTTPSINK")

	caCert, err := ioutil.ReadFile("/etc/ssl/certs/mdtp.pem")
	if err != nil {
		log.Info("Error Cert Read ", err)
	} else {
		log.Info("Cert:", caCert[0:20])
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// cert, err := tls.LoadX509KeyPair("client.crt", "client.key")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
				//Certificates: []tls.Certificate{cert},
			},
		},
	}

	request, err := http.NewRequest(has.method, has.endpoint, bytes.NewBuffer(encoded.Bytes))

	if err != nil {
		log.Info("Request ", request, " Error: ", err)

		return err
	} else {
		log.Info("Request ", request)
	}
	request.Header.Set("Content-Length", fmt.Sprintf("%d", encoded.Length()))
	res, err := client.Do(request)

	log.Info("Client Response: ", res, " error ", err)

	// res, err := http.DefaultClient.Do(request)
	if err != nil || res.StatusCode > 299 {
		log.SetFormatter(&log.JSONFormatter{
			FieldMap: log.FieldMap{
				log.FieldKeyMsg: "message",
			},
		})
		log.Warn("DS_EventMissed_AuditFailureResponse audit item : " + string(encoded.Bytes))
		return err
	}
	log.Info("Finished HTTPAuditSink")
	return res.Body.Close()
}

func (has *httpSink) Close() error {
	return nil
}
