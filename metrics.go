package fastgluemetrics

import (
	"bytes"
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
	NormalizeHTTPStatus   bool   // If multiple status codes like `400`,`404`,`413` are present, setting this to `true` will make them group under their parent category i.e. `4xx`.
	ExposeGoMetrics       bool   // Setting this to `true` would expose various `go_*` and `process_*` metrics.
	MatchedRoutePathParam string // If the value is set, the `path` variable in metric label will be the one used while registering the handler. If the value is unset, the original request path is used.
	ServiceName           string // Unique identifier for the service name.
}

// FastGlueMetrics represents the metrics instance.
type FastGlueMetrics struct {
	Opts    *Opts
	Metrics *metrics.Set
}

// NewMetrics initializes a new FastGlueMetrics instance with sane defaults.
func NewMetrics(g *fastglue.Fastglue, opts Opts) *FastGlueMetrics {
	m := &FastGlueMetrics{
		Opts: &Opts{
			NormalizeHTTPStatus:   true,
			ExposeGoMetrics:       false,
			MatchedRoutePathParam: g.MatchedRoutePathParam,
		},
		Metrics: metrics.NewSet(),
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
	m.Metrics.WritePrometheus(buf)
	if m.Opts.ExposeGoMetrics {
		metrics.WriteProcessMetrics(buf)
	}
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
		method = string(r.RequestCtx.Method())
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

	// Write the metrics.
	m.Metrics.GetOrCreateCounter(`requests_total{service="` + m.Opts.ServiceName +
		`", status="` + status + `", method="` + method + `", path="` + path + `"}`).Inc()

	m.Metrics.GetOrCreateHistogram(`request_duration_seconds{service="` + m.Opts.ServiceName +
		`", status="` + status + `", method="` + method + `", path="` + path + `"}`).UpdateDuration(start)

	m.Metrics.GetOrCreateHistogram(`response_size_bytes{service="` + m.Opts.ServiceName +
		`", status="` + status + `", method="` + method + `", path="` + path + `"}`).Update(size)

	return r
}
