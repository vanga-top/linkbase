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
