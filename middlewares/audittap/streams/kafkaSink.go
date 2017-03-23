package streams

//import (
//	. "github.com/containous/traefik/middlewares/audittap/audittypes"
//	"log"
//	"sync"
//	"github.com/Shopify/sarama"
//)
//
//type kafkaAuditSink struct {
//	topic    string
//	producer sarama.AsyncProducer
//	join     *sync.WaitGroup
//	render   Renderer
//}
//
//func NewKafkaSink(topic, endpoint string, renderer Renderer) (sink AuditSink, err error) {
//	config := sarama.NewConfig()
//	config.Producer.Return.Successes = false
//	producer, err := sarama.NewAsyncProducer([]string{endpoint}, config)
//	if err != nil {
//		panic(err)
//	}
//
//	kas := &kafkaAuditSink{topic, producer, &sync.WaitGroup{}, renderer}
//	kas.join.Add(1)
//
//	go func() {
//		// read errors and log them, until the producer is closed
//		for err := range producer.Errors() {
//			log.Errorf("Kafka: %v", err)
//		}
//		kas.join.Done()
//	}()
//
//	return kas, nil
//}
//
//func (kas *kafkaAuditSink) Audit(encoded Encoded) error {
//	message := &sarama.ProducerMessage{Topic: kas.topic, Value: encoded}
//	kas.producer.Input() <- message
//	return nil
//}
//
//func (kas *kafkaAuditSink) Close() error {
//	kas.producer.AsyncClose()
//	kas.join.Wait()
//	return nil
//}
