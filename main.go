package main

import (
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/valyala/fasthttp"
)

var (
	listenAddrs          = flag.String("listenAddrs", ":8098", "A list of TCP addresses to listen to HTTP requests. Leave empty if you don't need http")
	maxIdleUpstreamConns = flag.Int("maxIdleUpstreamConns", 50, "The maximum idle connections to upstream host")
	upstreamHost         = flag.String("upstreamHost", "www.google.com", "Upstream host to proxy data from. May include port in the form 'host:port'")
	upstreamProtocol     = flag.String("upstreamProtocol", "http", "Use this protocol when talking to the upstream")
	useClientRequestHost = flag.Bool("useClientRequestHost", false, "If set to true, then use 'Host' header from client requests in requests to upstream host. Otherwise use upstreamHost as a 'Host' header in upstream requests")
)

var upstreamClient *fasthttp.HostClient

func main() {
	flag.Parse()

	upstreamHostBytes = []byte(*upstreamHost)

	upstreamClient = &fasthttp.HostClient{
		Addr:     *upstreamHost,
		MaxConns: *maxIdleUpstreamConns,
	}

	h := requestHandler
	if *compress {
		h = fasthttp.CompressHandler(h)
	}

	if err := fasthttp.ListenAndServe(*listenAddrs, h); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

var keyPool sync.Pool

func requestHandler(ctx *fasthttp.RequestCtx) {

	var req fasthttp.Request
	var resp fasthttp.Response

	h := &ctx.Request.Header
	upstreamUrl := fmt.Sprintf("%s://%s%s", *upstreamProtocol, *upstreamHost, h.RequestURI())
	req.SetRequestURI(upstreamUrl)

	err := upstreamClient.Do(&req, &resp)
	if err != nil {
		ctx.Error("Bad Gateway", fasthttp.StatusBadGateway)
		return
	}

	v := keyPool.Get()
	if v == nil {
		v = make([]byte, 128)
	}
	key := v.([]byte)
	key = append(key[:0], getRequestHost(h)...)
	key = append(key, ctx.RequestURI()...)
	keyPool.Put(v)

	rh := &ctx.Response.Header

	ctx.Success(contentType, buf)
}

var upstreamHostBytes []byte

func getRequestHost(h *fasthttp.RequestHeader) []byte {
	if *useClientRequestHost {
		return h.Host()
	}
	return upstreamHostBytes
}