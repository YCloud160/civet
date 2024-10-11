package errors

import "encoding/json"

var (
	ErrRequestTimeout = NewError("", 502, "request timeout")
)

type Error struct {
	Code int32
	Desc string
	Id   string
}

func NewError(id string, code int32, desc string) error {
	err := &Error{
		Id:   id,
		Code: code,
		Desc: desc,
	}
	return err
}

func (err *Error) Error() string {
	if err == nil {
		return "nil"
	}
	bs, _ := json.Marshal(err)
	return string(bs)
}

func ParseError(err error) *Error {
	if e, ok := err.(*Error); ok {
		return e
	}
	e := &Error{}
	jerr := json.Unmarshal([]byte(err.Error()), e)
	if jerr != nil {
		e.Code = 9999
		e.Desc = err.Error()
	}
	return e
}
