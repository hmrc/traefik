package audittypes

// Encoder describes any type that can be encoded as an array of bytes
// in order to be sent as the key or value of a Kafka message. Length() is provided as
// an optimization, and must return the same as len() on the result of Encode().
// See https://godoc.org/github.com/Shopify/sarama#Encoder
type Encoder interface {
	Encode() ([]byte, error)
	Length() int
}

// Encoded holds encoded data
type Encoded struct {
	Bytes []byte
	Err   error
}

func (enc Encoded) Encode() ([]byte, error) {
	return enc.Bytes, enc.Err
}

func (enc Encoded) Length() int {
	return len(enc.Bytes)
}
