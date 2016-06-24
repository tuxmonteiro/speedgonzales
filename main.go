package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/valyala/fasthttp"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	listenAddrs          = flag.String("listenAddrs", ":8098", "A list of TCP addresses to listen to HTTP requests. Leave empty if you don't need http")
	maxIdleUpstreamConns = flag.Int("maxIdleUpstreamConns", 50, "The maximum idle connections to upstream host")
	upstreamHost         = flag.String("upstreamHost", "www.google.com", "Upstream host to proxy data from. May include port in the form 'host:port'")
	compress             = flag.Bool("compress", false, "Whether to enable transparent response compression")
)

var upstreamClient *http.Client

func main() {
	flag.Parse()

	upstreamClient = &http.Client{Timeout: 15 * time.Second}

	h := requestHandler
	if *compress {
		h = fasthttp.CompressHandler(h)
	}

	if err := fasthttp.ListenAndServe(*listenAddrs, h); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {

	headers := http.Header{}
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers.Add(string(key), string(value))
	})
	u, err := url.Parse(fmt.Sprint("http://", *upstreamHost, "/", string(ctx.Request.RequestURI())))
	if err != nil {
		ctx.Error("Bad Gateway", fasthttp.StatusBadGateway)
		log.Print(err)
		return
	}
	req := &http.Request{
		Method:        string(ctx.Method()),
		URL:           u,
		Proto:         map[bool]string{true: "HTTP/1.1", false: "HTTP/1.0"}[ctx.Request.Header.IsHTTP11()],
		ProtoMinor:    map[bool]int{true: 1, false: 0}[ctx.Request.Header.IsHTTP11()],
		Header:        headers,
		Body:          ioutil.NopCloser(bytes.NewReader(ctx.Request.Body())),
		ContentLength: int64(ctx.Request.Header.ContentLength()),
		Host:          string(ctx.Request.Header.Host()),
	}

	resp, err := upstreamClient.Do(req)
	if err != nil {
		ctx.Error("Bad Gateway", fasthttp.StatusBadGateway)
		log.Print(err)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		ctx.Response.Header.Add(k, strings.Join(v, ","))
	}
	io.Copy(ctx.Response.BodyWriter(), resp.Body)
}
