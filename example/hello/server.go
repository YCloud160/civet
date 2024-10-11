package main

import (
	"context"
	"fmt"
	"github.com/YCloud/civet"
	"github.com/YCloud/civet/example/hello/model"
)

type HelloServer struct{}

func (s *HelloServer) SayHello(ctx context.Context, req *model.HelloReq) (*model.HelloResp, error) {
	fmt.Println("say hello to ", req.Name)
	return &model.HelloResp{Message: "Hello " + req.Name}, nil
}

func (s *HelloServer) Dispatch(ctx context.Context, impl any, enc civet.Encoder, method string, in []byte) (out []byte, err error) {
	switch method {
	case "SayHello":
		req := model.HelloReq{}
		err := enc.Unmarshal(in, &req)
		if err != nil {
			return nil, err
		}
		obj := impl.(*HelloServer)
		resp, err := obj.SayHello(ctx, &req)
		if err != nil {
			return nil, err
		}
		return enc.Marshal(resp)
	default:
		return nil, fmt.Errorf("unknown method")
	}
}

func main() {
	obj := &HelloServer{}
	civet.AddRPCServant("hello", obj, obj.Dispatch)
	if err := civet.Run(); err != nil {
		fmt.Println(err)
		return
	}
}
