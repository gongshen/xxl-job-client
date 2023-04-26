package executor

import (
	"github.com/valyala/fasthttp"
	"time"
)

func NewHttpServer(handler fasthttp.RequestHandler) *fasthttp.Server {
	return &fasthttp.Server{
		Handler:      handler,
		Concurrency:  1024,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		TCPKeepalive: true,
	}
}
