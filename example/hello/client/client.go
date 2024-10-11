package main

import (
	"context"
	"fmt"
	"github.com/YCloud/civet"
	"github.com/YCloud/civet/example/hello/model"
	"sync"
)

func main() {
	client := civet.NewClient("hello", civet.WithClientOptionEndpoint(&civet.Endpoint{
		IP:   "127.0.0.1",
		Port: "10010",
	}))
	wg := sync.WaitGroup{}
	n := 10
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func(i int) {
			defer wg.Done()
			req := &model.HelloReq{Name: fmt.Sprintf("civet %d", i)}
			resp := &model.HelloResp{}
			err := client.Call(context.TODO(), "SayHello", "", req, resp)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(resp)
		}(i)
	}
	wg.Wait()
}
