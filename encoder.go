package civet

import (
	"github.com/YCloud/civet/encoder/jsonencoder"
)

type Encoder interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, addr any) error
	Name() string
}

var encoderMap = map[string]Encoder{}

func init() {
	RegisterEncoder(jsonencoder.NewJSONEncoder())
}

func RegisterEncoder(enc Encoder) {
	if enc == nil {
		panic("encoder is nil")
	}
	if enc.Name() == "" {
		panic("encoder name is empty")
	}
	encoderMap[enc.Name()] = enc
}

func GetEncoder(name string) Encoder {
	return encoderMap[name]
}
