package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/valyala/fasthttp"
	"REDACTED/commons/fastglue"
	fastgluemetrics "REDACTED/commons/fastglue-metrics"
)

var (
	fakeResponse = make([]byte, 1024*1000*1)
)

func main() {
	// Initialize fastglue.
	g := fastglue.NewGlue()
	// Initialise fastglue-metrics exporter.
	exporter := fastgluemetrics.NewMetrics(g, fastgluemetrics.Opts{
		ExposeGoMetrics:       true,
		NormalizeHTTPStatus:   true,
		ServiceName:           "dummy",
		MatchedRoutePathParam: g.MatchedRoutePathParam,
	})
	// Register handlers.
	g.GET("/", func(r *fastglue.Request) error {
		return r.SendEnvelope("Welcome to dummy-app metrics. Visit /metrics.")
	})
	g.GET("/fake", func(r *fastglue.Request) error {
		r.RequestCtx.Write(fakeResponse)
		return nil
	})
	g.GET("/slow/:user/ping", func(r *fastglue.Request) error {
		sleep := 0.5 + rand.Float64()*1.75
		time.Sleep(time.Duration(sleep) * 1000 * time.Millisecond)
		return r.SendEnvelope("Sleeping slow respo")
	})
	g.GET("/bad/:user", func(r *fastglue.Request) error {
		status := [9]int{300, 400, 413, 500, 417, 404, 402, 503, 502}
		return r.SendErrorEnvelope(status[rand.Intn(9)], "oops", nil, "")
	})
	// Expose the registered metrics at `/metrics` path.
	g.GET("/metrics", exporter.HandleMetrics)
	// HTTP server.
	s := &fasthttp.Server{
		Name:                 "metrics",
		ReadTimeout:          time.Millisecond * 3000,
		WriteTimeout:         time.Millisecond * 6000,
		MaxKeepaliveDuration: time.Millisecond * 5000,
		MaxRequestBodySize:   50000,
		ReadBufferSize:       50000,
	}
	log.Println("starting server on :6090")
	if err := g.ListenAndServe("0.0.0.0:6090", "", s); err != nil {
		log.Println("error starting server:", err)
	}
}
