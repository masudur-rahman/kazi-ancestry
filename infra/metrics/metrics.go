// Package metrics provides Prometheus instrumentation for the server: HTTP
// request metrics, business counters, and scrape-time domain gauges. Everything
// registers on the default registry and is exposed via Handler().
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/masudur-rahman/kazi-ancestry/models"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kazi_http_requests_total",
		Help: "Total HTTP requests by method, route pattern, and status code.",
	}, []string{"method", "route", "code"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kazi_http_request_duration_seconds",
		Help:    "HTTP request latency by method, route pattern, and status code.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "code"})

	httpInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kazi_http_requests_in_flight",
		Help: "In-flight HTTP requests.",
	})

	logins = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kazi_logins_total",
		Help: "OAuth login outcomes.",
	}, []string{"result"})

	suggestionsSubmitted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kazi_suggestions_submitted_total",
		Help: "Suggestions submitted by contributors.",
	})

	suggestionsResolved = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kazi_suggestions_resolved_total",
		Help: "Suggestions resolved by admins.",
	}, []string{"action"})
)

// Handler serves the Prometheus exposition endpoint.
func Handler() http.Handler { return promhttp.Handler() }

// Middleware records per-request HTTP metrics. The route label is the chi route
// pattern (e.g. /api/v1/people/{id}), not the raw path, to keep cardinality low.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpInFlight.Inc()
		defer httpInFlight.Dec()

		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		code := strconv.Itoa(ww.Status())
		httpRequests.WithLabelValues(r.Method, route, code).Inc()
		httpDuration.WithLabelValues(r.Method, route, code).Observe(time.Since(start).Seconds())
	})
}

// LoginResult records an OAuth login outcome: "success", "denied", or "error".
func LoginResult(result string) { logins.WithLabelValues(result).Inc() }

// SuggestionSubmitted records a submitted suggestion.
func SuggestionSubmitted() { suggestionsSubmitted.Inc() }

// SuggestionResolved records a resolved suggestion: action "approved" or "rejected".
func SuggestionResolved(action string) { suggestionsResolved.WithLabelValues(action).Inc() }

// RegisterDomain registers scrape-time gauges derived from the data layer. The
// closures are evaluated on every scrape, so this package needs no service deps.
func RegisterDomain(peopleCount func() (int, error), listSuggestions func() ([]models.Suggestion, error)) {
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "kazi_people_total",
		Help: "Number of people in the tree.",
	}, func() float64 {
		n, err := peopleCount()
		if err != nil {
			return 0
		}
		return float64(n)
	})

	prometheus.MustRegister(&suggestionsCollector{list: listSuggestions})
}

// suggestionsCollector emits kazi_suggestions_total{status} on each scrape.
type suggestionsCollector struct {
	list func() ([]models.Suggestion, error)
}

var suggestionsDesc = prometheus.NewDesc(
	"kazi_suggestions_total",
	"Suggestions by status.",
	[]string{"status"}, nil,
)

func (c *suggestionsCollector) Describe(ch chan<- *prometheus.Desc) { ch <- suggestionsDesc }

func (c *suggestionsCollector) Collect(ch chan<- prometheus.Metric) {
	counts := map[string]float64{"pending": 0, "approved": 0, "rejected": 0}
	if list, err := c.list(); err == nil {
		for _, s := range list {
			counts[s.Status]++
		}
	}
	for status, n := range counts {
		ch <- prometheus.MustNewConstMetric(suggestionsDesc, prometheus.GaugeValue, n, status)
	}
}
