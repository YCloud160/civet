package jsonencoder

import "encoding/json"

type jsonEncoder struct{}

func NewJSONEncoder() *jsonEncoder {
	return &jsonEncoder{}
}

func (*jsonEncoder) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (*jsonEncoder) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (*jsonEncoder) Name() string {
	return "json"
}
