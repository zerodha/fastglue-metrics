package fastgluemetrics

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"REDACTED/commons/fastglue"
)

const (
	// Key to store the current time in `ctx.UserValue`
	latencyKey = "latency_probe"
)

// Register various time series metrics.
var (
	labelRequestsTotal = `requests_total{service="%s", status="%s", method="%s", path="%s"}`
	labelResponseSize  = `response_size_bytes{service="%s", status="%s", method="%s", path="%s"}`
	labelRequestTime   = `request_duration_seconds{service="%s", status="%s", method="%s", path="%s"}`
)

// Opts represents configuration properties for metrics exposition.
type Opts struct {
	NormalizeHTTPStatus   bool
	ExposeGoMetrics       bool
	MatchedRoutePathParam string
	ServiceName           string
}

// FastGlueMetrics represents the metrics instance.
type FastGlueMetrics struct {
	Opts *Opts
}

// NewMetrics initializes a new FastGlueMetrics instance with sane defaults.
func NewMetrics(g *fastglue.Fastglue, opts Opts) *FastGlueMetrics {
	m := &FastGlueMetrics{
		Opts: &Opts{
			NormalizeHTTPStatus:   true,
			ExposeGoMetrics:       false,
			MatchedRoutePathParam: g.MatchedRoutePathParam,
		},
	}
	if opts != (Opts{}) {
		m.Opts = &opts
	}
	// Register middlewares.
	g.Before(m.before)
	g.After(m.after)
	return m
}

// HandleMetrics returns the metric data response.
func (m *FastGlueMetrics) HandleMetrics(r *fastglue.Request) error {
	buf := new(bytes.Buffer)
	metrics.WritePrometheus(buf, m.Opts.ExposeGoMetrics)
	return r.SendBytes(200, "text/plain; version=0.0.4", buf.Bytes())
}

func (m *FastGlueMetrics) before(r *fastglue.Request) *fastglue.Request {
	r.RequestCtx.SetUserValue(latencyKey, time.Now())
	return r
}

func (m *FastGlueMetrics) after(r *fastglue.Request) *fastglue.Request {
	var (
		path   = ""
		status = strconv.Itoa(r.RequestCtx.Response.StatusCode())
		start  = r.RequestCtx.UserValue(latencyKey).(time.Time)
		method = r.RequestCtx.Method()
		size   = float64(len(r.RequestCtx.Response.Body()))
	)
	// MatchedRoutePathParam stores the actual path before string interpolation by the router.
	// This is useful if you want to prevent high cardinality in labels.
	// For example, for a path `/orders/:userid/get` the number of metric series would be directly proportional
	// to all the unique `userid` hitting that endpoint. In order to prevent such high label cardinality, the raw
	// path string which is set to register the handler, is used for the metric label `path`.
	if m.Opts.MatchedRoutePathParam != "" {
		path = r.RequestCtx.UserValue(m.Opts.MatchedRoutePathParam).(string)
	} else {
		path = string(r.RequestCtx.URI().Path())
	}
	// NormalizeHTTPStatus groups arbitary status codes by their cateogry.
	// For example 400,417,413 will be grouped as 4xx.
	if m.Opts.NormalizeHTTPStatus {
		status = string(status[0]) + "xx"
	}
	// Construct metric labels.
	requestsTotalDesc := fmt.Sprintf(labelRequestsTotal, m.Opts.ServiceName, status, method, path)
	requestsTimeDesc := fmt.Sprintf(labelRequestTime, m.Opts.ServiceName, status, method, path)
	responseSizeDesc := fmt.Sprintf(labelResponseSize, m.Opts.ServiceName, status, method, path)
	// Dynamically create metrics if a new label has come up or reuse the existing
	// metric object if the label is same.
	metrics.GetOrCreateCounter(requestsTotalDesc).Inc()
	metrics.GetOrCreateHistogram(requestsTimeDesc).UpdateDuration(start)
	metrics.GetOrCreateHistogram(responseSizeDesc).Update(size)
	return r
}
