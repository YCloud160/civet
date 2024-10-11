package civet

import (
	"context"
	"fmt"
	"github.com/YCloud/civet/internal/config"
	"github.com/YCloud/civet/tlog"
	"net"
	"net/http"
)

type HttpServerOption func(srv *httpServer)

func WithHttpInterceptors(interceptors ...HttpInterceptor) HttpServerOption {
	return func(srv *httpServer) {
		srv.interceptors = append(srv.interceptors, interceptors...)
	}
}

type httpServer struct {
	http.Server

	name             string
	cfg              *config.ServantConf
	endpoint         *Endpoint
	interceptors     []HttpInterceptor
	unaryInterceptor HttpInterceptor
}

func newHttpServer(name string, handler http.Handler, opts ...HttpServerOption) *httpServer {
	cfg := config.GetServantConf(name)
	srv := &httpServer{
		name: name,
		cfg:  cfg,
	}

	for _, opt := range opts {
		opt(srv)
	}

	srv.Addr = fmt.Sprintf("%s:%s", cfg.IP, cfg.Port)
	srv.Handler = handler
	return srv
}

func (srv *httpServer) Start() error {
	srv.unaryInterceptor = buildHttpInterceptor(srv.interceptors...)
	lis, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		app.wg.Done()
		app.startErr = err
		return err
	}
	app.wg.Done()
	tlog.Info("start servant", tlog.Any("servant", srv.Name()), tlog.Any("endpoint", srv.Endpoint().IPPort()))
	return srv.Serve(lis)
}

func (srv *httpServer) Stop() error {
	return srv.Shutdown(context.TODO())
}

func (srv *httpServer) Name() string {
	return srv.name
}

func (srv *httpServer) Endpoint() *Endpoint {
	return srv.endpoint
}

func (srv *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.unaryInterceptor(w, r, srv.Handler.ServeHTTP)
}
