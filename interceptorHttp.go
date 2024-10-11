package civet

import (
	"net/http"
)

type HttpInterceptor func(w http.ResponseWriter, r *http.Request, handler http.HandlerFunc)

func buildHttpInterceptor(httpInterceptors ...HttpInterceptor) HttpInterceptor {
	interceptors := make([]HttpInterceptor, 0)
	interceptors = append(interceptors, traceHttpInterceptor)
	interceptors = append(interceptors, httpInterceptors...)
	interceptors = append(interceptors, recoverHttpInterceptor)
	return chainHttpInterceptor(interceptors...)
}

func chainHttpInterceptor(httpInterceptors ...HttpInterceptor) HttpInterceptor {
	if len(httpInterceptors) == 0 {
		return nil
	} else if len(httpInterceptors) == 1 {
		return httpInterceptors[0]
	} else {
		return chainChainHttpHandler(httpInterceptors)
	}
}

func chainChainHttpHandler(interceptors []HttpInterceptor) HttpInterceptor {
	return func(w http.ResponseWriter, r *http.Request, handler http.HandlerFunc) {
		interceptors[0](w, r, getChainHttpHandler(interceptors, 0, handler))
	}
}

func getChainHttpHandler(interceptors []HttpInterceptor, curr int, finalHandler http.HandlerFunc) http.HandlerFunc {
	if curr == len(interceptors)-1 {
		return finalHandler
	}
	return func(w http.ResponseWriter, r *http.Request) {
		interceptors[curr+1](w, r, getChainHttpHandler(interceptors, curr+1, finalHandler))
	}
}

func traceHttpInterceptor(w http.ResponseWriter, req *http.Request, handler http.HandlerFunc) {
	ctx := req.Context()
	// TODO
	req = req.WithContext(ctx)
	handler(w, req)
}

func recoverHttpInterceptor(w http.ResponseWriter, req *http.Request, handler http.HandlerFunc) {
	defer func() {
		if r := recover(); r != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}()
	handler(w, req)
}
