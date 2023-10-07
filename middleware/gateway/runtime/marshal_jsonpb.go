package runtime

import (
	"google.golang.org/protobuf/encoding/protojson"
	"io"
)

type JSONPb struct {
	protojson.MarshalOptions
	protojson.UnmarshalOptions
}

func (J JSONPb) Marshal(v interface{}) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (J JSONPb) Unmarshal(data []byte, v interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (J JSONPb) NewDecoder(r io.Reader) Decoder {
	//TODO implement me
	panic("implement me")
}

func (J JSONPb) NewEncoder(w io.Writer) Encoder {
	//TODO implement me
	panic("implement me")
}

func (J JSONPb) ContentType(v interface{}) string {
	//TODO implement me
	panic("implement me")
}
