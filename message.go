package civet

import (
	"context"
	"fmt"
)

var ErrMaxBodySize = fmt.Errorf("max body size")

const (
	MaxBodyLen int32 = 1<<30 - 8

	HeaderLen = 5

	MessageHeaderMask = 1 << 7
	MessageCodeMask   = 1 << 6
	MessageFlagMask   = 1<<6 - 1
)

// 消息标识
type MessageFlag uint8

const (
	// ping 需要返回响应
	MessageFlag_Ping MessageFlag = 0
	// ping 响应返回
	MessageFlag_PingResp MessageFlag = 1
	// 保活
	MessageFlag_KeepAlive MessageFlag = 2
	// 消息推送，不需要返回
	MessageFlag_Push MessageFlag = 3
	// 数据请求
	MessageFlag_Req MessageFlag = 4
	// 数据请求返回
	MessageFlag_Resp MessageFlag = 5
	//MessageFlag_Ping MessageFlag = 6
	//MessageFlag_Ping MessageFlag = 7
)

type Message struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	Req    *Request

	Encode Encoder

	Resp *Response
}

type Request struct {
	StreamId int32
	Flag     MessageFlag
	Route    string
	Header   map[string]string
	Body     []byte
}

type Response struct {
	StreamId int32
	Flag     MessageFlag
	Code     int32
	CodeDesc string
	Header   map[string]string
	Body     []byte
}
