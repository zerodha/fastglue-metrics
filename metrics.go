package fastgluemetrics

import (
	"bytes"
	"fmt"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"REDACTED/commons/fastglue"
)

// Register various time series.
// Time series name may contain labels in Prometheus format - see below.
var (
	labelRequestsTotal = `requests_total{}`
	labelResponseSize  = `response_size_bytes{status="%d", method="%s", path="%s"}`
	labelRequestTime   = `request_duration_seconds{status="%d", method="%s", path="%s"}`
)

// Opts represent initialising config options for metric exposition
type Opts struct {
	NormalizeHTTPStatus   bool
	ExposeGoMetrics       bool
	MatchedRoutePathParam string
}

type FastGlueMetrics struct {
	Opts *Opts
}

// NewMetrics blah
func NewMetrics(g *fastglue.Fastglue, opts *Opts) *FastGlueMetrics {
	m := &FastGlueMetrics{
		Opts: &Opts{
			NormalizeHTTPStatus: true,
			ExposeGoMetrics:     false,
		},
	}
	if opts != nil {
		m.Opts = opts
	}
	g.Before(m.before)
	g.After(m.after)
	return m
}

// HandleMetrics blah
func (m *FastGlueMetrics) HandleMetrics(r *fastglue.Request) error {
	buf := new(bytes.Buffer)
	metrics.WritePrometheus(buf, m.Opts.ExposeGoMetrics)
	return r.SendBytes(200, "text/plain; version=0.0.4", buf.Bytes())
}

func (m *FastGlueMetrics) before(r *fastglue.Request) *fastglue.Request {
	r.RequestCtx.SetUserValue("latency_probe", time.Now())
	return r
}

func (m *FastGlueMetrics) after(r *fastglue.Request) *fastglue.Request {
	var (
		start  = r.RequestCtx.UserValue("latency_probe").(time.Time)
		status = r.RequestCtx.Response.StatusCode()
		method = r.RequestCtx.Method()
		size   = float64(len(r.RequestCtx.Response.Body()))
	)
	path := ""
	// string(r.RequestCtx.Request.URI().PathOriginal())
	if m.Opts.MatchedRoutePathParam != "" {
		path = r.RequestCtx.UserValue(m.Opts.MatchedRoutePathParam).(string)
	} else {
		path = string(r.RequestCtx.URI().Path())
	}
	requestsTotalDesc := fmt.Sprintf(labelRequestsTotal)
	requestsTimeDesc := fmt.Sprintf(labelRequestTime, status, method, path)
	responseSizeDesc := fmt.Sprintf(labelResponseSize, status, method, path)
	metrics.GetOrCreateCounter(requestsTotalDesc).Inc()
	metrics.GetOrCreateHistogram(requestsTimeDesc).UpdateDuration(start)
	metrics.GetOrCreateHistogram(responseSizeDesc).Update(size)
	return r
}
