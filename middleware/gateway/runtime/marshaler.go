package runtime

import "io"

type Marshaler interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
	NewDecoder(r io.Reader) Decoder
	NewEncoder(w io.Writer) Encoder
	ContentType(v interface{}) string
}

type Decoder interface {
	Decode(v interface{}) error
}

type Encoder interface {
	Encode(v interface{}) error
}

type DecoderFunc func(v interface{}) error

func (f DecoderFunc) Decode(v interface{}) error {
	return f(v)
}

type EncoderFunc func(v interface{}) error

func (f EncoderFunc) Encode(v interface{}) error {
	return f(v)
}

type Delimited interface {
	Delimiter() []byte
}
