package civet

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	errors2 "github.com/YCloud/civet/errors"
	"github.com/YCloud/civet/internal/config"
	"github.com/YCloud/civet/meta"
	"net"
	"sync"
	"sync/atomic"
)

var ErrBadConn = errors.New("bad connection")

type clientCallOptions struct {
	enc Encoder
}

type ClientCallOption func(*clientCallOptions)

func WithClientCallOptionEncoder(enc Encoder) ClientCallOption {
	return func(o *clientCallOptions) {
		o.enc = enc
	}
}

type ClientOption func(*Client)

func WithClientDefaultEncoder(enc Encoder) ClientOption {
	return func(client *Client) {
		client.enc = enc
	}
}

func WithClientOptionEndpoint(endpoints ...*Endpoint) ClientOption {
	return func(client *Client) {
		client.endpoints = endpoints
	}
}

func WithClientInterceptors(interceptors ...ClientInterceptor) ClientOption {
	return func(client *Client) {
		client.interceptors = append(client.interceptors, interceptors...)
	}
}

type Client struct {
	mux       sync.Mutex
	reqId     atomic.Int32
	service   string
	endpoints []*Endpoint
	idx       int
	enc       Encoder
	pools     map[string]*clientConnPool
	reqData   sync.Map
	recvCh    chan []byte

	cfg *config.ClientConf

	interceptors     []ClientInterceptor
	unaryInterceptor ClientInterceptor
}

func NewClient(service string, options ...ClientOption) *Client {
	client := &Client{
		service:      service,
		endpoints:    make([]*Endpoint, 0),
		pools:        make(map[string]*clientConnPool),
		recvCh:       make(chan []byte, 10000),
		interceptors: make([]ClientInterceptor, 0),
		cfg:          config.GetClientConf(),
	}

	for _, option := range options {
		option(client)
	}

	if client.enc == nil {
		client.enc = GetEncoder(client.cfg.EncoderName)
	}

	client.unaryInterceptor = buildClientInterceptor(client.interceptors...)
	for _, endpoint := range client.endpoints {
		ipport := endpoint.IPPort()
		client.pools[ipport] = newClientConnPool(client, ipport, client.cfg.MaxConnNum)
	}

	go client.recvProcess()

	return client
}

func (client *Client) Call(ctx context.Context, method string, ipport string, req, rsp any, options ...ClientCallOption) error {
	callOptions := &clientCallOptions{}
	for _, option := range options {
		option(callOptions)
	}

	reqHeader, ok := meta.FromMetaContextReqContext(ctx)
	if !ok {
		reqHeader = make(map[string]string)
	}

	enc, err := client.getEncoder(callOptions, reqHeader)
	if err != nil {
		return err
	}
	reqBytes, err := enc.Marshal(req)
	if err != nil {
		return err
	}
	reqHeader = meta.CopyHeader(reqHeader)
	reqHeader[meta.ContentType] = enc.Name()

	reqMsg := &Request{
		StreamId: client.reqId.Add(1),
		Flag:     MessageFlag_Req,
		Route:    client.getRoute(method),
		Header:   reqHeader,
		Body:     reqBytes,
	}

	interceptorFun := client.unaryInterceptor
	if interceptorFun != nil {
		err = interceptorFun(ctx, ipport, reqMsg, enc, rsp, client.invoker)
	} else {
		err = client.invoker(ctx, ipport, reqMsg, enc, rsp)
	}
	return err
}

func (client *Client) invoker(ctx context.Context, ipport string, reqMsg *Request, enc Encoder, rsp any) error {
	clientConn, err := client.getConn(ctx, ipport)
	if err != nil {
		return err
	}
	reqMsgBytes, err := MarshalRequest(reqMsg)
	if err != nil {
		return err
	}
	rspChan := make(chan *Response, 1)
	client.reqData.Store(reqMsg.StreamId, rspChan)
	defer client.reqData.Delete(reqMsg.StreamId)
	defer close(rspChan)

	_, err = clientConn.conn.Write(reqMsgBytes)
	if err != nil {
		fmt.Println(err)
		return err
	}

	select {
	case <-ctx.Done():
		return errors2.ErrRequestTimeout
	case rspMsg := <-rspChan:
		err = nil
		if rspMsg.Code > 0 && len(rspMsg.CodeDesc) > 0 {
			err = errors2.NewError(client.service, rspMsg.Code, rspMsg.CodeDesc)
		}
		if rspMsg.Code == 0 && len(rspMsg.CodeDesc) > 0 {
			err = errors2.NewError(client.service, 502, rspMsg.CodeDesc)
		}
		if err != nil {
			return err
		}
		meta.NewMetaContextWithRespContext(ctx, rspMsg.Header)
		return enc.Unmarshal(rspMsg.Body, rsp)
	}
}

func (client *Client) recvProcess() {
	for {
		select {
		case bs := <-client.recvCh:
			rspMsg, err := ParserResponse(bs)
			if err != nil {
				continue
			}
			switch rspMsg.Flag {
			case MessageFlag_Resp:
				if val, ok := client.reqData.Load(rspMsg.StreamId); ok {
					val.(chan *Response) <- rspMsg
				}
			}
		}
	}
}

func (client *Client) getEncoder(callOptions *clientCallOptions, reqHeader map[string]string) (Encoder, error) {
	enc := callOptions.enc
	if enc == nil {
		enc = client.enc
		if encName, ok := reqHeader[meta.ContentType]; ok {
			enc = GetEncoder(encName)
		}
	}
	if enc == nil {
		return nil, errors.New("encoder is nil")
	}
	return enc, nil
}

func (client *Client) getRoute(method string) string {
	return fmt.Sprintf("%s/%s", client.service, method)
}

func (client *Client) getConn(ctx context.Context, ipport string) (*clientConn, error) {
	client.mux.Lock()
	defer client.mux.Unlock()

	if len(ipport) != 0 {
		pool, ok := client.pools[ipport]
		if !ok {
			return nil, ErrBadConn
		}
		return pool.get()
	} else {
		for i := 0; i < 3; i++ {
			endpoint := client.endpoints[client.idx%len(client.endpoints)]
			client.idx++

			pool, ok := client.pools[endpoint.IPPort()]
			if ok {
				conn, err := pool.get()
				if err == nil {
					return conn, nil
				}
			}
		}
	}
	return nil, ErrBadConn
}

type clientConnPool struct {
	client     *Client
	mux        sync.Mutex
	addr       string
	conn       []*clientConn
	idx        int
	maxConnNum int // 最大连接数
}

func newClientConnPool(client *Client, addr string, maxConnNum int) *clientConnPool {
	pool := &clientConnPool{
		client:     client,
		addr:       addr,
		conn:       make([]*clientConn, 0, maxConnNum),
		maxConnNum: maxConnNum,
	}
	return pool
}

func (pool *clientConnPool) get() (*clientConn, error) {
	pool.mux.Lock()
	defer pool.mux.Unlock()

	if len(pool.conn) == 0 {
		pool.tryConnect()
	}

	if len(pool.conn) == 0 {
		return nil, ErrBadConn
	}

	conn := pool.conn[pool.idx%len(pool.conn)]
	pool.idx++

	if len(pool.conn) < pool.maxConnNum {
		go pool.tryConnectLocked()
	}

	return conn, nil
}

func (pool *clientConnPool) tryConnectLocked() {
	pool.mux.Lock()
	defer pool.mux.Unlock()
	pool.tryConnect()
}

func (pool *clientConnPool) tryConnect() {
	if len(pool.conn) >= pool.maxConnNum {
		return
	}
	conn, err := net.Dial("tcp", pool.addr)
	if err != nil {
		fmt.Println("连接失败", err)
		return
	}
	c := newClientConn(conn, pool.client)
	go c.recv()
	pool.conn = append(pool.conn, c)
}

type clientConn struct {
	client *Client
	conn   net.Conn
}

func newClientConn(conn net.Conn, client *Client) *clientConn {
	c := &clientConn{conn: conn, client: client}
	return c
}

func (c *clientConn) recv() {
	buf := make([]byte, 8192)
	currentBuf := make([]byte, 0, 8192)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			return
		}
		currentBuf = append(currentBuf, buf[:n]...)
		for {
			body, n, stat := c.readBody(currentBuf)
			if stat == ReadStat_Continue {
				break
			} else if stat == ReadStat_Full {
				currentBuf = currentBuf[n:]
				c.client.recvCh <- body
			} else {
				return
			}
		}
	}
}

func (c *clientConn) readBody(buf []byte) ([]byte, int, ReadStat) {
	if len(buf) <= 4 {
		return nil, 0, ReadStat_Continue
	}
	length := int32(binary.LittleEndian.Uint32(buf))
	if int32(len(buf)) < length {
		return nil, 0, ReadStat_Continue
	}
	body := buf[4:length]
	return body, int(length), ReadStat_Full
}
