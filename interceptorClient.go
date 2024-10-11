package civet

import (
	"context"
)

type ClientInvoker func(ctx context.Context, ipport string, reqMsg *Request, enc Encoder, rsp any) error

type ClientInterceptor func(ctx context.Context, ipport string, reqMsg *Request, enc Encoder, rsp any, invoker ClientInvoker) error

func buildClientInterceptor(clientInterceptors ...ClientInterceptor) ClientInterceptor {
	interceptors := make([]ClientInterceptor, 0)
	interceptors = append(interceptors, traceClientInterceptor)
	interceptors = append(interceptors, clientInterceptors...)
	interceptors = append(interceptors, recoverClientInterceptor)
	return chainClientInterceptor(interceptors...)
}

func chainClientInterceptor(clientInterceptors ...ClientInterceptor) ClientInterceptor {
	if len(clientInterceptors) == 0 {
		return nil
	} else if len(clientInterceptors) == 1 {
		return clientInterceptors[0]
	} else {
		return chainUnaryClientInterceptor(clientInterceptors)
	}
}

func chainUnaryClientInterceptor(interceptors []ClientInterceptor) ClientInterceptor {
	return func(ctx context.Context, ipport string, reqMsg *Request, enc Encoder, rsp any, invoker ClientInvoker) error {
		return interceptors[0](ctx, ipport, reqMsg, enc, rsp, getChainClientInterceptor(interceptors, 0, invoker))
	}
}

func getChainClientInterceptor(interceptors []ClientInterceptor, curr int, finalInvoker ClientInvoker) ClientInvoker {
	if curr == len(interceptors)-1 {
		return finalInvoker
	}
	return func(ctx context.Context, ipport string, reqMsg *Request, enc Encoder, rsp any) error {
		return interceptors[curr+1](ctx, ipport, reqMsg, enc, rsp, getChainClientInterceptor(interceptors, curr+1, finalInvoker))
	}
}

func traceClientInterceptor(ctx context.Context, ipport string, reqMsg *Request, enc Encoder, rsp any, invoker ClientInvoker) error {
	return invoker(ctx, ipport, reqMsg, enc, rsp)
}

func recoverClientInterceptor(ctx context.Context, ipport string, reqMsg *Request, enc Encoder, rsp any, invoker ClientInvoker) error {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	return invoker(ctx, ipport, reqMsg, enc, rsp)
}
