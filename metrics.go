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

// NewMetrics blah
func NewMetrics(g *fastglue.Fastglue) {
	g.Before(before)
	g.After(after)
}

// HandleMetrics blah
func HandleMetrics(r *fastglue.Request) error {
	buf := new(bytes.Buffer)
	metrics.WritePrometheus(buf, false)
	return r.SendBytes(200, "text/plain; version=0.0.4", buf.Bytes())
}

func before(r *fastglue.Request) *fastglue.Request {
	r.RequestCtx.SetUserValue("latency_probe", time.Now())
	return r
}

func after(r *fastglue.Request) *fastglue.Request {
	var (
		start  = r.RequestCtx.UserValue("latency_probe").(time.Time)
		status = r.RequestCtx.Response.StatusCode()
		method = r.RequestCtx.Method()
		path   = string(r.RequestCtx.Request.URI().PathOriginal())
		size   = float64(len(r.RequestCtx.Response.Body()))
	)
	requestsTotalDesc := fmt.Sprintf(labelRequestsTotal)
	requestsTimeDesc := fmt.Sprintf(labelRequestTime, status, method, path)
	responseSizeDesc := fmt.Sprintf(labelResponseSize, status, method, path)
	metrics.GetOrCreateCounter(requestsTotalDesc).Inc()
	metrics.GetOrCreateHistogram(requestsTimeDesc).UpdateDuration(start)
	metrics.GetOrCreateHistogram(responseSizeDesc).Update(size)
	return r
}
