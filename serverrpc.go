package civet

import (
	"context"
	"encoding/binary"
	errors2 "errors"
	"fmt"
	"github.com/YCloud/civet/errors"
	"github.com/YCloud/civet/internal/config"
	"github.com/YCloud/civet/meta"
	"github.com/YCloud/civet/tlog"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

type ReadStat uint8

const (
	ReadStat_Err ReadStat = 1 + iota
	ReadStat_Full
	ReadStat_Continue
)

type ServerOption func(srv *rpcServer)

func WithServerInterceptors(interceptors ...ServerInterceptor) ServerOption {
	return func(srv *rpcServer) {
		srv.interceptors = append(srv.interceptors, interceptors...)
	}
}

type rpcServer struct {
	name     string
	impl     any
	dispatch Dispatch
	cfg      *config.ServantConf
	endpoint *Endpoint

	interceptors     []ServerInterceptor
	unaryInterceptor ServerInterceptor

	reqQueue chan struct{}
	reqNum   atomic.Int32

	mu         sync.Mutex
	isShutdown atomic.Bool
	listen     net.Listener
	conns      map[*serverConn]struct{}
}

func newRpcServer(name string, impl any, dispatch Dispatch, opts ...ServerOption) *rpcServer {
	cfg := config.GetServantConf(name)
	srv := &rpcServer{
		name:     name,
		dispatch: dispatch,
		impl:     impl,
		cfg:      cfg,
		conns:    make(map[*serverConn]struct{}),
	}
	srv.endpoint = &Endpoint{
		IP:   cfg.IP,
		Port: cfg.Port,
	}

	for _, opt := range opts {
		opt(srv)
	}

	srv.reqQueue = make(chan struct{}, srv.cfg.MaxRequestNum)
	return srv
}

func (srv *rpcServer) Start() error {
	srv.unaryInterceptor = buildServerInterceptor(srv.interceptors...)
	listen, err := net.Listen("tcp", fmt.Sprintf(":%s", srv.cfg.Port))
	if err != nil {
		app.wg.Done()
		app.startErr = err
		return err
	}
	if srv.listen != nil {
		srv.listen.Close()
	}
	srv.listen = listen
	app.wg.Done()
	tlog.Info("start servant", tlog.Any("servant", srv.Name()), tlog.Any("endpoint", srv.Endpoint().IPPort()))
	return srv.accept()
}

func (srv *rpcServer) Stop() error {
	srv.isShutdown.Store(true)
	return nil
}

func (srv *rpcServer) Name() string {
	return srv.name
}

func (srv *rpcServer) Endpoint() *Endpoint {
	return srv.endpoint
}

func (srv *rpcServer) accept() error {
	defer srv.close()
	for {
		conn, err := srv.listen.Accept()
		if err != nil {
			return err
		}
		sc := srv.newServerConn(conn)
		go sc.send()
		go sc.recv()
	}
}

func (srv *rpcServer) close() {
	srv.listen.Close()
	srv.mu.Lock()
	for sc := range srv.conns {
		delete(srv.conns, sc)
	}
	srv.mu.Unlock()
}

func (srv *rpcServer) newServerConn(conn net.Conn) *serverConn {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if srv.cfg.ReadBufSize > 0 {
			tcpConn.SetReadBuffer(int(srv.cfg.ReadBufSize))
		}
		if srv.cfg.WriteBufSize > 0 {
			tcpConn.SetWriteBuffer(int(srv.cfg.WriteBufSize))
		}
	}
	sendChanCap := 1
	if srv.cfg.MaxRequestNum%10 > 1 {
		sendChanCap = int(srv.cfg.MaxRequestNum % 10)
	}
	sc := &serverConn{
		srv:       srv,
		conn:      conn,
		closeChan: make(chan struct{}),
		sendChan:  make(chan *Message, sendChanCap),
	}
	srv.mu.Lock()
	srv.conns[sc] = struct{}{}
	srv.mu.Unlock()
	return sc
}

type serverConn struct {
	conn      net.Conn
	srv       *rpcServer
	isClose   atomic.Bool
	sendChan  chan *Message
	closeChan chan struct{}
}

func (sc *serverConn) close() {
	sc.srv.mu.Lock()
	if sc.isClose.Load() == true {
		sc.srv.mu.Unlock()
		return
	}
	sc.isClose.Store(true)
	delete(sc.srv.conns, sc)
	sc.srv.mu.Unlock()

	sc.conn.Close()
	close(sc.closeChan)
}

func (sc *serverConn) recv() {
	defer sc.close()

	readBuf := make([]byte, 8192)
	currentBuf := make([]byte, 0, 8192)
	for {
		n, err := sc.conn.Read(readBuf)
		if err != nil {
			if errors2.Is(err, io.EOF) || errors2.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("读取数据错误，err:%v\n", err)
			return
		}
		currentBuf = append(currentBuf, readBuf[:n]...)
		for {
			body, n, stat := sc.readBody(currentBuf)
			if stat == ReadStat_Continue {
				break
			} else if stat == ReadStat_Full {
				currentBuf = currentBuf[n:]
				sc.handle(body)
			} else {
				return
			}
		}
	}
}

func (sc *serverConn) readBody(buf []byte) ([]byte, int, ReadStat) {
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

func (sc *serverConn) handle(pkg []byte) {
	req, err := ParserRequest(pkg)
	if err != nil {
		return
	}
	switch req.Flag {
	case MessageFlag_Req:
		sc.invokeRequest(req)
	case MessageFlag_Ping:

	}
}

func (sc *serverConn) invokeRequest(req *Request) {
	sc.srv.reqNum.Add(1)
	defer sc.srv.reqNum.Add(-1)

	msg := &Message{
		Ctx:  context.TODO(),
		Req:  req,
		Resp: &Response{StreamId: req.StreamId, Flag: MessageFlag_Resp},
	}
	i := strings.LastIndex(req.Route, "/")
	method := req.Route[i+1:]
	contentType := req.Header[meta.ContentType]
	msg.Encode = GetEncoder(contentType)
	if msg.Encode == nil {
		msg.Resp.Code = 402
		msg.Resp.CodeDesc = "content type error"
		sc.sendChan <- msg
		return
	}

	cfg := sc.srv.cfg
	if cfg.ReqTimeout > 0 {
		msg.Ctx, msg.Cancel = context.WithTimeout(msg.Ctx, cfg.ReqTimeout)
	} else {
		msg.Ctx, msg.Cancel = context.WithCancel(msg.Ctx)
	}
	msg.Ctx = meta.NewMetaContextWithReqContext(msg.Ctx, msg.Req.Header)

	select {
	case <-msg.Ctx.Done():
		msg.Resp.Code = 504
		msg.Resp.CodeDesc = "request timeout"
		sc.sendChan <- msg
		return
	case sc.srv.reqQueue <- struct{}{}:
		defer func() {
			<-sc.srv.reqQueue
		}()
	}
	go func() {
		var (
			out []byte
			err error
		)
		intercept := sc.srv.unaryInterceptor
		if intercept != nil {
			out, err = intercept(msg.Ctx, sc.srv.impl, msg.Encode, method, msg.Req.Body, sc.srv.dispatch)
		} else {
			out, err = sc.srv.dispatch(msg.Ctx, sc.srv.impl, msg.Encode, method, msg.Req.Body)
		}
		if err != nil {
			e := errors.ParseError(err)
			msg.Resp.Code = e.Code
			msg.Resp.CodeDesc = e.Desc
		} else {
			msg.Resp.Body = out
			msg.Resp.Header, _ = meta.FromMetaContextRespContext(msg.Ctx)
		}
		msg.Cancel()
	}()

	select {
	case <-msg.Ctx.Done():
		if msg.Resp.Code == 0 && len(msg.Resp.CodeDesc) == 0 && len(msg.Resp.Body) == 0 {
			msg.Resp.Code = 504
			msg.Resp.CodeDesc = "request timeout"
		}
		sc.sendChan <- msg
	}
}

func (sc *serverConn) send() {
	for {
		select {
		case <-sc.closeChan:
			return
		case msg := <-sc.sendChan:
			body, err := MarshalResponse(msg.Resp)
			if err != nil {
				log.Printf("send body failed err:%v\n", err)
			} else {
				sc.conn.Write(body)
			}
		}
	}
}
