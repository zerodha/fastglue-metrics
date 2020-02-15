package main

import (
	"log"
	"time"

	"github.com/valyala/fasthttp"
	"REDACTED/commons/fastglue"
	metrics "REDACTED/fastglue-metrics"
)

func main() {
	// Initialize fastglue.
	g := fastglue.NewGlue()
	metrics.NewMetrics(g)
	// Handlers.
	g.GET("/", func(r *fastglue.Request) error {
		// return r.SendErrorEnvelope(500, "oops", nil, "")
		return r.SendEnvelope("Welcome to Metrics")
	})
	g.GET("/slow/:user/ping", func(r *fastglue.Request) error {
		time.Sleep(2000 * time.Millisecond)
		// return r.SendErrorEnvelope(500, "oops", nil, "")
		return r.SendEnvelope("Sleeping slow respo")
	})
	// Expose the registered metrics at `/metrics` path.
	g.GET("/metrics", metrics.HandleMetrics)
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
	log.Println("Bye")
}
