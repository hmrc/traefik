package streams

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"os"
	"io"

	"github.com/beeker1121/goque"
	"github.com/containous/traefik/middlewares/audittap/configuration"
	"github.com/containous/traefik/middlewares/audittap/encryption"
	atypes "github.com/containous/traefik/middlewares/audittap/types"
	log "github.com/sirupsen/logrus"
)

type httpAuditSinkAsync struct {
	cli       *http.Client
	messages  chan atypes.Encoded
	producers []*httpProducerAsync
	q         *goque.Queue
	enc       encryption.Encrypter
}

// NewHttpSink returns an AuditSink for sending messages to Datastream service.
// A connection is made to the specified endpoint and a number of Producers
// each backed by a channel are created, ready to send messages.
func NewHTTPSinkAsync(config *configuration.AuditSink, messageChan chan atypes.Encoded) (sink AuditSink, err error) {
	var client = CreateClient()

	producers := make([]*httpProducerAsync, 0)
	q, err := NewQueue(config.DiskStorePath)
	if err != nil {
		return nil, err
	}

	enc, err := encryption.NewEncrypter(config.EncryptSecret)
	if err != nil {
		return nil, err
	}
	for i := 0; i < config.NumProducers; i++ {
		p, _ := newHttpProducerAsync(client, config.Endpoint, messageChan, q, enc)
		producers = append(producers, p)
	}

	aas := &httpAuditSinkAsync{cli: client, producers: producers, messages: messageChan, q: q, enc: enc}

	return aas, nil
}

func CreateClient() (*http.Client) {
	var certPath = os.Getenv("CERTIFICATEPATH")
	var client *http.Client

	if len(certPath) > 0 {
		caCert, err := ioutil.ReadFile(certPath)
		if err != nil {
			log.Error("Error Cert Read ", err)
			os.Exit(1)
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

func (aas *httpAuditSinkAsync) Audit(encoded atypes.Encoded) error {
	select {
	case aas.messages <- encoded:
	default:
		handleFailedMessage(encoded, "channel full", aas.enc)
	}
	return nil
}

func (aas *httpAuditSinkAsync) Close() error {
	for _, p := range aas.producers {
		p.stop <- true
	}
	aas.q.Close()
	return nil
}

type httpProducerAsync struct {
	cli      *http.Client
	endpoint string
	messages chan atypes.Encoded
	q        *goque.Queue
	stop     chan bool
	enc      encryption.Encrypter
}

func newHttpProducerAsync(client *http.Client, endpoint string, messages chan atypes.Encoded, q *goque.Queue, enc encryption.Encrypter) (*httpProducerAsync, error) {
	stop := make(chan bool)
	producer := &httpProducerAsync{cli: client, endpoint: endpoint, messages: messages, q: q, stop: stop, enc: enc}
	go producer.audit()
	go producer.publish()
	return producer, nil
}

func (p *httpProducerAsync) audit() {
	for {
		encoded := <-p.messages
		_, err := p.q.EnqueueObject(encoded)
		if err != nil {
			handleFailedMessage(encoded, "enqueue failed", p.enc)
		}
	}
}

func constructRequest(endpoint string, encoded atypes.Encoded) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(encoded.Bytes))
	if err != nil {
		log.Warn("DS_EventMissed_AuditFailureResponse audit item : " + string(encoded.Bytes)) //TODO is that correct?
		return nil, err
	}
	request.Header.Set("Content-Length", fmt.Sprintf("%d", encoded.Length()))
	return request, nil
}

func sendRequest(cli *http.Client, encoded atypes.Encoded, request *http.Request) {
	log.SetFormatter(&log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyMsg: "message",
		},
	})

	res, err := cli.Do(request)

	if err != nil || res.StatusCode > 299 {
		log.Warn("DS_EventMissed_AuditFailureResponse audit item : " + string(encoded.Bytes))
		return
	}
	defer res.Body.Close()
	// close the http body before making a new http request: https://golang.cafe/blog/how-to-reuse-http-connections-in-go.html
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
}

func (p *httpProducerAsync) publish() {
	for {
		select {
		case <-p.stop:
			return
		default:
			item, err := p.q.Dequeue()
			if err != nil {
				if err == goque.ErrEmpty {
					time.Sleep(2 * time.Millisecond)
					continue
				}
				// now? nothing to see here ... Should only happen if reference to goque.q is "closed"
				log.Error(err)
				continue
			}
			var encoded atypes.Encoded
			if err = item.ToObject(&encoded); err != nil {
				// well, that didn't work
				log.Error(err)
			}

			select {
			case <-p.stop:
				// we've been asked to stop prior to publication: re-enqueue the audit message in disk queue
				p.q.EnqueueObject(encoded)
				return
			default:
				req, err := constructRequest(p.endpoint, encoded)
				if err != nil {
					log.Error(err)
					return
				}
				sendRequest(p.cli, encoded, req)
			}
		}
	}
}
