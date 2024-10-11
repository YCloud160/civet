package meta

import "context"

type metaKey struct{}

type Meta struct {
	ReqContext  map[string]string
	RespContext map[string]string
}

func getMetaContext(ctx context.Context) (*Meta, bool) {
	meta, ok := ctx.Value(&metaKey{}).(*Meta)
	return meta, ok
}

func NewMetaContextWithReqContext(ctx context.Context, reqContext map[string]string) context.Context {
	meta, ok := getMetaContext(ctx)
	if !ok {
		meta = &Meta{
			ReqContext: reqContext,
		}
	} else {
		for k, v := range reqContext {
			meta.ReqContext[k] = v
		}
	}
	return context.WithValue(ctx, &metaKey{}, meta)
}

func FromMetaContextReqContext(ctx context.Context) (map[string]string, bool) {
	meta, ok := getMetaContext(ctx)
	if ok {
		return meta.ReqContext, true
	}
	return nil, false
}

func NewMetaContextWithRespContext(ctx context.Context, respContext map[string]string) context.Context {
	meta, ok := getMetaContext(ctx)
	if !ok {
		meta = &Meta{
			RespContext: respContext,
		}
	} else {
		for k, v := range respContext {
			meta.RespContext[k] = v
		}
	}
	return context.WithValue(ctx, &metaKey{}, meta)
}

func FromMetaContextRespContext(ctx context.Context) (map[string]string, bool) {
	meta, ok := getMetaContext(ctx)
	if ok {
		return meta.RespContext, true
	}
	return nil, false
}

func CopyHeader(header map[string]string) map[string]string {
	mp := make(map[string]string)
	for k, v := range header {
		mp[k] = v
	}
	return mp
}
