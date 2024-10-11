package civet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"strings"
)

const (
	maxHeaderSize = 1<<24 - 1
)

var ErrNotEnoughBytes = errors.New("not enough bytes")
var ErrMaxBytes = errors.New("max bytes")

func ParserRequest(bs []byte) (*Request, error) {
	l := len(bs)
	r := 0
	if r+4 >= l {
		return nil, ErrNotEnoughBytes
	}
	req := &Request{}
	req.StreamId = int32(binary.LittleEndian.Uint32(bs[r:]))
	r += 4
	if r+1 >= l {
		return nil, ErrNotEnoughBytes
	}
	flag := bs[r]
	r += 1
	req.Flag = MessageFlag(flag & MessageFlagMask)
	if r+2 >= l {
		return nil, ErrNotEnoughBytes
	}
	routeSize := binary.LittleEndian.Uint16(bs[r:])
	r += 2
	if len(bs[r:]) < int(routeSize) {
		return nil, ErrNotEnoughBytes
	}
	req.Route = string(bs[r : r+int(routeSize)])
	r += int(routeSize)
	if r+3 >= l {
		return nil, ErrNotEnoughBytes
	}
	headerSize := int(bs[r]) | int(bs[r+1])<<8 | int(bs[r+2])<<16
	r += 3
	if len(bs[r:]) < headerSize {
		return nil, ErrNotEnoughBytes
	}
	req.Header = parserHeader(string(bs[r : r+headerSize]))
	r += headerSize
	req.Body = bs[r:]
	return req, nil
}

func MarshalRequest(req *Request) ([]byte, error) {
	routeSize := len(req.Route)
	if routeSize > math.MaxUint16 {
		return nil, ErrMaxBytes
	}
	header := marshalHeader(req.Header)
	headerSize := len(header)
	if headerSize > maxHeaderSize {
		return nil, ErrMaxBytes
	}

	n := 14 + len(req.Route) + len(header) + len(req.Body)
	b := make([]byte, n)
	w := 0
	binary.LittleEndian.PutUint32(b[w:], uint32(n))
	w += 4
	binary.LittleEndian.PutUint32(b[w:], uint32(req.StreamId))
	w += 4
	b[w] = byte(req.Flag & MessageFlagMask)
	w += 1
	binary.LittleEndian.PutUint16(b[w:], uint16(routeSize))
	w += 2
	copy(b[w:], req.Route)
	w += routeSize
	b[w] = byte(headerSize)
	b[w+1] = byte(headerSize >> 8)
	b[w+2] = byte(headerSize >> 16)
	w += 3
	copy(b[w:], header)
	w += headerSize
	copy(b[w:], req.Body)
	return b, nil
}

func ParserResponse(bs []byte) (*Response, error) {
	l := len(bs)
	r := 0
	if r+4 >= l {
		return nil, ErrNotEnoughBytes
	}
	rsp := &Response{}
	rsp.StreamId = int32(binary.LittleEndian.Uint32(bs[r:]))
	r += 4
	if r+1 >= l {
		return nil, ErrNotEnoughBytes
	}
	flag := bs[r]
	r += 1
	rsp.Flag = MessageFlag(flag & MessageFlagMask)
	codeBit := flag & MessageCodeMask
	if codeBit > 0 {
		if r+4 >= l {
			return nil, ErrNotEnoughBytes
		}
		rsp.Code = int32(binary.LittleEndian.Uint32(bs[r:]))
		r += 4
		if r+2 >= l {
			return nil, ErrNotEnoughBytes
		}
		codeDescSize := binary.LittleEndian.Uint16(bs[r:])
		r += 2
		if len(bs[r:]) < int(codeDescSize) {
			return nil, ErrNotEnoughBytes
		}
		rsp.CodeDesc = string(bs[r : r+int(codeDescSize)])
		r += int(codeDescSize)
	}
	if r+3 >= l {
		return nil, ErrNotEnoughBytes
	}
	headerSize := int(bs[r]) | int(bs[r+1])<<8 | int(bs[r+2])<<16
	r += 3
	if len(bs[r:]) < headerSize {
		return nil, ErrNotEnoughBytes
	}
	rsp.Header = parserHeader(string(bs[r : r+headerSize]))
	r += headerSize
	rsp.Body = bs[r:]
	return rsp, nil
}

func MarshalResponse(rsp *Response) ([]byte, error) {
	codeDescSize := len(rsp.CodeDesc)
	if codeDescSize > math.MaxUint16 {
		return nil, ErrMaxBytes
	}
	header := marshalHeader(rsp.Header)
	headerSize := len(header)
	if headerSize > maxHeaderSize {
		return nil, ErrMaxBytes
	}

	n := 12 + len(header) + len(rsp.Body)
	if rsp.Code > 0 {
		n = n + 6 + codeDescSize
	}

	b := make([]byte, n)
	w := 0
	binary.LittleEndian.PutUint32(b[w:], uint32(n))
	w += 4
	binary.LittleEndian.PutUint32(b[w:], uint32(rsp.StreamId))
	w += 4
	if rsp.Code > 0 {
		b[w] = byte(rsp.Flag&MessageFlagMask) | byte(MessageCodeMask)
		w += 1
		binary.LittleEndian.PutUint32(b[w:], uint32(rsp.Code))
		w += 4
		binary.LittleEndian.PutUint16(b[w:], uint16(codeDescSize))
		w += 2
		copy(b[w:], rsp.CodeDesc)
		w += codeDescSize
	} else {
		b[w] = byte(rsp.Flag & MessageFlagMask)
		w += 1
	}
	b[w] = byte(headerSize)
	b[w+1] = byte(headerSize >> 8)
	b[w+2] = byte(headerSize >> 16)
	w += 3
	copy(b[w:], header)
	w += headerSize
	copy(b[w:], rsp.Body)
	return b, nil
}

func parserHeader(s string) map[string]string {
	mp := make(map[string]string)
	var (
		k, v string
	)
	for {
		if s == "" {
			break
		}
		k, s, _ = strings.Cut(s, "&")
		k, v, _ = strings.Cut(k, "=")
		mp[k] = v
	}
	return mp
}

func marshalHeader(mp map[string]string) string {
	buf := bytes.NewBuffer([]byte{})
	ks := make([]string, 0, len(mp))
	for k := range mp {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for i, k := range ks {
		if i > 0 {
			buf.WriteByte('&')
		}
		v := mp[k]
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(v)
	}
	return buf.String()
}
