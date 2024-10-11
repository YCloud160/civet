package civet

import (
	"context"
	"fmt"
	"github.com/YCloud/civet/tlog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Dispatch func(ctx context.Context, impl any, enc Encoder, method string, in []byte) (out []byte, err error)

type Server interface {
	Start() error
	Stop() error
	Name() string
	Endpoint() *Endpoint
}

type application struct {
	servantList []Server
	wg          sync.WaitGroup
	startErr    error
	stopChan    chan struct{}
}

var app *application

func init() {
	app = &application{
		servantList: make([]Server, 0),
		stopChan:    make(chan struct{}),
	}
}

func AddHTTPServant(name string, handler http.Handler, opts ...HttpServerOption) {
	srv := newHttpServer(name, handler, opts...)
	addServant(srv)
}

func AddRPCServant(name string, impl any, dispatch Dispatch, options ...ServerOption) {
	srv := newRpcServer(name, impl, dispatch, options...)
	addServant(srv)
}

func addServant(server Server) {
	app.servantList = append(app.servantList, server)
}

func Run() error {

	for _, srv := range app.servantList {
		app.wg.Add(1)
		go func(srv Server) {
			if err := srv.Start(); err != nil {
				return
			}
		}(srv)
	}
	app.wg.Wait()
	if app.startErr != nil {
		os.Exit(1)
		return app.startErr
	}

	return mainLoop()
}

func mainLoop() error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	timer := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-timer.C:
			keepalive()
		case v := <-signals:
			tlog.Info("服务中断")
			switch v {
			case syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT:
				stop()
			}
			return fmt.Errorf("%v", v)
		case <-app.stopChan:
			stop()
			return nil
		}
	}
}

func keepalive() {

}

func stop() {
	tlog.Info("stop service begin")
	for _, srv := range app.servantList {
		tlog.Info("stop servant", tlog.Any("servant", srv.Name()), tlog.Any("endpoint", srv.Endpoint().IPPort()))
		err := srv.Stop()
		if err != nil {
			tlog.Error("stop servant error", tlog.Any("err", err))
		}
	}
	tlog.Info("stop service end")
	tlog.Flush()
}
