package main

import (
	"flag"
	"github.com/valyala/fasthttp"
	"log"
	"time"
)

var (
	listenAddrs          = flag.String("listenAddrs", ":8098", "A list of TCP addresses to listen to HTTP requests. Leave empty if you don't need http")
	maxIdleUpstreamConns = flag.Int("maxIdleUpstreamConns", 50, "The maximum idle connections to upstream host")
	upstreamHost         = flag.String("upstreamHost", "www.google.com", "Upstream host to proxy data from. May include port in the form 'host:port'")
	compress             = flag.Bool("compress", false, "Whether to enable transparent response compression")
)

var upstreamClient *fasthttp.HostClient

func main() {
	flag.Parse()

	upstreamClient = newClient()

	h := requestHandler
	if *compress {
		h = fasthttp.CompressHandler(h)
	}

	if err := fasthttp.ListenAndServe(*listenAddrs, h); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {

	err := upstreamClient.DoTimeout(&ctx.Request, &ctx.Response, 15 * time.Second)
	if err != nil {
		ctx.Error("Bad Gateway", fasthttp.StatusBadGateway)
		log.Print(err)
		return
	}

}

func newClient() *fasthttp.HostClient {
	return &fasthttp.HostClient{
		Addr:                          *upstreamHost,
		MaxConns:                      *maxIdleUpstreamConns,
		DisableHeaderNamesNormalizing: true,
	}
}
