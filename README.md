# fastglue-metrics

![grafana-screenshot](screenshots/grafana.png)

## Overview [![Go Reference](https://pkg.go.dev/badge/github.com/zerodha/fastglue-metrics.svg)](https://pkg.go.dev/github.com/zerodha/fastglue-metrics) [![Zerodha Tech](https://zerodha.tech/static/images/github-badge.svg)](https://zerodha.tech)

This package provides an abstraction to collect HTTP metrics from any Golang application using the package [fastglue](https://github.com/zerodha/fastglue). It uses [before-after](https://github.com/zerodha/fastglue/tree/master/examples/before-after) middlewares to collect metrics about the request such as:

- Count of HTTP requests.
- Latency of each request.
- Response size.

The metrics collection is inspired from [RED](https://grafana.com/blog/2018/08/02/the-red-method-how-to-instrument-your-services/) principles of application monitoring. The components of this monitoring philosophy are:

- **Rate** (the number of requests per second)
- **Errors** (the number of those requests that are failing)
- **Duration** (the amount of time those requests take)

All the metrics are grouped by the following labels:

- `status`: (HTTP Status Code)
- `path`: (Request URI)
- `method`: (HTTP Method)

## Usage

`go get github.com/zerodha/fastglue-metrics`

To start collecting metrics, simply initialise the metric exporter with:

```go
package main

import (
    fastgluemetrics "github.com/zerodha/fastglue-metrics"
)

// Initialize fastglue.
g := fastglue.NewGlue()
// Initialise fastglue-metrics exporter.
exporter := fastgluemetrics.NewMetrics(g, fastgluemetrics.Opts{
    ExposeGoMetrics:       true,
    NormalizeHTTPStatus:   false,
    ServiceName: "dummy",
    MatchedRoutePathParam: g.MatchedRoutePathParam,
})
// Expose the registered metrics at `/metrics` path.
g.GET("/metrics", exporter.HandleMetrics)
```

### Additional Options

You can configure options to configure the behaviour of exporter using `fastgluemetrics.Opts`.
To see a fully working example, you can check [examples/main](examples/main.go).

### Exporting Custom App Metrics

In case your app needs to export custom app related metrics, you can modify the 
following example.

```go
// StatsManager is a struct that will hold your custom stats.
type StatsManager struct {
	Stats       map[string]int64
	ServiceName string
	sync.RWMutex
}

// NewStats returns an instance of StatsManager.
func NewStats(serviceName string) *StatsManager {
	if serviceName == "" {
		serviceName = "myapp"
	}

	return &StatsManager{
		Stats:       map[string]int64{},
		ServiceName: serviceName,
	}
}

// PromFormatter writes the value in prometheus format with the service name.
func (s *StatsManager) PromFormatter(b *bytes.Buffer, key string, val int64) {
	fmt.Fprintf(b, `%s{service="%s"} %d`, key, s.ServiceName, val)
	fmt.Fprintln(b)
}

// HandleMetrics returns a handler which exports stats.
func (app *App) HandleMetrics(g *fastglue.Fastglue) fastglue.FastRequestHandler {
    // Initialize the fastglue exporter
	exporter := fastgluemetrics.NewMetrics(g, fastgluemetrics.Opts{
		ExposeGoMetrics:       true,
		NormalizeHTTPStatus:   true,
		ServiceName:           "veto",
		MatchedRoutePathParam: g.MatchedRoutePathParam,
	})

	return func(r *fastglue.Request) error {
		app.Stats.RLock()
		defer app.Stats.RUnlock()

		buf := new(bytes.Buffer)

        // Write the metrics to the buffer
		exporter.Metrics.WritePrometheus(buf)
		metrics.WriteProcessMetrics(buf)

		for _, k := range sortedKeys(app.Stats.Stats) {
            // Format and write to the buffer
			app.Stats.PromFormatter(buf, fmt.Sprintf("count_%s", k), app.Stats.Stats[k])
		}

		return r.SendBytes(200, "text/plain; version=0.0.4", buf.Bytes())
	}
}
```

## Configuration

`metrics.Options` takes in additional configurtion to customise the behaviour of exposition.

- **ServiceName**: Unique identifier for the service name.

- **NormalizeHTTPStatus**: If multiple status codes like `400`,`404`,`413` are present, setting this to `true` will make them group under their parent category i.e. `4xx`.

- **ExposeGoMetrics**: Setting this to `true` would expose various `go_*` and `process_*` metrics.

- **MatchedRoutePathParam**: If the value is set, the `path` variable in metric label will be the one used while registering the handler. If the value is unset, the original request path is used.

The value is exposed by `fastglue` as `Fastglue.MatchedRoutePathParam`.

## Notes

### Label Cardinality

If your application has dynamic endpoints, which make use of the [`Named Params` ](https://github.com/buaazp/fasthttprouter#named-parameters), you **must** set this value in order to keep label cardinality in check. If this value is not set, then a new metric will be created for each dynamic value of the named parameter, thus impacting the performance of downstream monitoring systems.

For example, for a route `/orders/:userid/fetch`, you don't want a million timeseries labels to be created for each user.`fasthttprouter` would set the value of matched path in `ctx.UserValue` with a **key**. This setting is the value of that key, which is exposed in `fastglue` package with the variable name: `MatchedRoutePathParam`.

### Metrics Collection Library

This package uses [VictoriaMetrics/metrics](https://github.com/VictoriaMetrics/metrics) which is an extremely lightweight alternative to the official Prometheus [client library](https://github.com/prometheus/client_golang). The official library pulls a lot of external dependencies and has complex features which are not really needed for this use case.

Besides being performant, `VM/metrics` has several improvements and optimisations on how a `Histogram` metric is constructed. For more details, you can read [this](https://medium.com/@valyala/improving-histogram-usability-for-prometheus-and-grafana-bc7e5df0e350) blog post.

## LICENSE

See [LICENSE](./LICENSE).
