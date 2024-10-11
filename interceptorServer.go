package civet

import (
	"context"
)

type ServerInterceptor func(ctx context.Context, impl any, enc Encoder, method string, in []byte, dispatch Dispatch) ([]byte, error)

func buildServerInterceptor(httpInterceptors ...ServerInterceptor) ServerInterceptor {
	interceptors := make([]ServerInterceptor, 0)
	interceptors = append(interceptors, traceServerInterceptor)
	interceptors = append(interceptors, httpInterceptors...)
	interceptors = append(interceptors, recoverServerInterceptor)
	return chainServerInterceptor(interceptors...)
}

func chainServerInterceptor(httpInterceptors ...ServerInterceptor) ServerInterceptor {
	if len(httpInterceptors) == 0 {
		return nil
	} else if len(httpInterceptors) == 1 {
		return httpInterceptors[0]
	} else {
		return chainUnaryServerHandler(httpInterceptors)
	}
}

func chainUnaryServerHandler(interceptors []ServerInterceptor) ServerInterceptor {
	return func(ctx context.Context, impl any, enc Encoder, method string, in []byte, dispatch Dispatch) ([]byte, error) {
		return interceptors[0](ctx, impl, enc, method, in, getChainServerHandler(interceptors, 0, dispatch))
	}
}

func getChainServerHandler(interceptors []ServerInterceptor, curr int, finalDispatch Dispatch) Dispatch {
	if curr == len(interceptors)-1 {
		return finalDispatch
	}
	return func(ctx context.Context, impl any, enc Encoder, method string, in []byte) ([]byte, error) {
		return interceptors[curr+1](ctx, impl, enc, method, in, getChainServerHandler(interceptors, curr+1, finalDispatch))
	}
}

func traceServerInterceptor(ctx context.Context, impl any, enc Encoder, method string, in []byte, dispatch Dispatch) ([]byte, error) {
	return dispatch(ctx, impl, enc, method, in)
}

func recoverServerInterceptor(ctx context.Context, impl any, enc Encoder, method string, in []byte, dispatch Dispatch) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {

		}
	}()
	return dispatch(ctx, impl, enc, method, in)
}
