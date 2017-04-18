package streams

import (
	"testing"
	"time"

	"bytes"
	"github.com/assembla/cony"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/middlewares/audittap/audittypes"
	"github.com/containous/traefik/types"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"strings"
)

type ConyClientTestImpl struct {
	Endpoint     string
	Message      amqp.Publishing
	Declarer     *TestDeclarer
	PublishCount int
	CloseCount   int
}

type TestDeclarer struct {
	Name, Kind                            string
	Durable, AutoDelete, Internal, NoWait bool
	Args                                  amqp.Table
}

func (d *TestDeclarer) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{}, nil
}

func (d *TestDeclarer) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	d.Name = name
	d.Kind = kind
	d.Durable = durable
	d.AutoDelete = autoDelete
	d.Internal = internal
	d.NoWait = noWait
	d.Args = args
	return nil
}

func (d *TestDeclarer) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return nil
}

func (c *ConyClientTestImpl) Declare(d []cony.Declaration) {
	if len(d) > 0 {
		c.Declarer = &TestDeclarer{}
		decl := d[0]
		decl(c.Declarer)
	}
}

func (c *ConyClientTestImpl) Errors() <-chan error {
	return make(chan error)
}

func (c *ConyClientTestImpl) Blocking() <-chan amqp.Blocking {
	return make(chan amqp.Blocking)
}

func (c *ConyClientTestImpl) Publish(pub amqpConyPublisher) {
	c.PublishCount++
}

func (c *ConyClientTestImpl) Close() {
	c.CloseCount++
}

func (c *ConyClientTestImpl) Loop() bool {
	return true
}

type ConyPublisherTestImpl struct {
	CallCount int
	Pub       amqp.Publishing
}

func (p *ConyPublisherTestImpl) Publish(pub amqp.Publishing) error {
	p.CallCount++
	p.Pub = pub
	return nil
}

func (p *ConyPublisherTestImpl) Cancel() {
}

func (p *ConyPublisherTestImpl) GetConyPublisher() *cony.Publisher {
	return nil
}

func TestAmqpSink(t *testing.T) {
	testClientImpl := &ConyClientTestImpl{Endpoint: "endpoint"}

	NewConyClient = func(endpoint string) amqpConyClient {
		return testClientImpl
	}

	testPublisherImpl := &ConyPublisherTestImpl{}

	NewConyPublisher = func(exchange string) amqpConyPublisher {
		return testPublisherImpl
	}

	var config = types.AuditSink{
		Endpoint:    "foo",
		Destination: "bar",
	}

	messages := make(chan audittypes.Encoded, 1)
	sink, err := NewAmqpSink(&config, messages)
	assert.NoError(t, err)

	newAmqpProducer(testClientImpl, "bar", messages)

	err = sink.Audit(encodedJSONSample)
	assert.NoError(t, err)

	err = sink.Close()
	assert.NoError(t, err)

	assert.Equal(t, testClientImpl.Endpoint, "endpoint")
	assert.Equal(t, testClientImpl.Declarer.Name, "bar")
	assert.Equal(t, testClientImpl.Declarer.Kind, "topic")
	assert.Equal(t, testClientImpl.Declarer.AutoDelete, false)
	assert.Equal(t, testClientImpl.Declarer.Durable, true)

	assert.Equal(t, testClientImpl.CloseCount, 1)
	assert.Equal(t, testClientImpl.PublishCount, 1)

	//TODO: Is there a better way to do this?
	for i := 0; len(messages) != 0 && i < 100; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	assert.Equal(t, testPublisherImpl.CallCount, 1)
	assert.Equal(t, testPublisherImpl.Pub.Body, encodedJSONSample.Bytes)
	assert.Equal(t, testPublisherImpl.Pub.DeliveryMode, amqp.Persistent)
}

func TestAmqpSinkFull(t *testing.T) {
	testClientImpl := &ConyClientTestImpl{Endpoint: "endpoint"}

	buf := new(bytes.Buffer)

	log.SetOutput(buf)

	NewConyClient = func(endpoint string) amqpConyClient {
		return testClientImpl
	}

	testPublisherImpl := &ConyPublisherTestImpl{}

	NewConyPublisher = func(exchange string) amqpConyPublisher {
		return testPublisherImpl
	}

	var config = types.AuditSink{
		Endpoint:    "foo",
		Destination: "bar",
	}

	messages := make(chan audittypes.Encoded, 0)
	sink, err := NewAmqpSink(&config, messages)
	assert.NoError(t, err)

	newAmqpProducer(testClientImpl, "bar", messages)

	err = sink.Audit(encodedJSONSample)
	assert.NoError(t, err)

	err = sink.Close()
	assert.NoError(t, err)

	assert.Equal(t, testPublisherImpl.CallCount, 0)

	bufStr := buf.String()

	assert.True(t, strings.Contains(bufStr, `level=error msg="Message not delivered to MQ because channel full body: [1,2,3]"`))
}
