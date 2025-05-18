// Package metrics provides functionality to collect and expose metrics for the application
package metrics

import (
	"context"
	"strconv"
	
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// Metrics holds all the metrics collectors for the application
type Metrics struct {
	registry           *prometheus.Registry
	requestsTotal      *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	dbQueryDuration    *prometheus.HistogramVec
	activeConnections  prometheus.Gauge
	walletOperations   *prometheus.CounterVec
	walletBalanceTotal *prometheus.GaugeVec
}

// NewMetrics creates and registers all application metrics
func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	
	// Request metrics
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	
	// Database metrics
	dbQueryDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)
	
	// Connection metrics
	activeConnections := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)
	
	// Business metrics
	walletOperations := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_operations_total",
			Help: "Total number of wallet operations",
		},
		[]string{"operation", "status"},
	)
	
	walletBalanceTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_balance_total",
			Help: "Total wallet balance per user",
		},
		[]string{"user_id", "currency"},
	)

	// Register all metrics
	registry.MustRegister(
		requestsTotal,
		requestDuration,
		dbQueryDuration,
		activeConnections,
		walletOperations,
		walletBalanceTotal,
	)

	return &Metrics{
		registry:           registry,
		requestsTotal:      requestsTotal,
		requestDuration:    requestDuration,
		dbQueryDuration:    dbQueryDuration,
		activeConnections:  activeConnections,
		walletOperations:   walletOperations,
		walletBalanceTotal: walletBalanceTotal,
	}
}

// RecordRequest records metrics about an HTTP request
func (m *Metrics) RecordRequest(method, path string, status int) {
	m.requestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
}

// ObserveRequestDuration records the duration of an HTTP request
func (m *Metrics) ObserveRequestDuration(method, path string, duration float64) {
	m.requestDuration.WithLabelValues(method, path).Observe(duration)
}

// ObserveDBQueryDuration records the duration of a database query
func (m *Metrics) ObserveDBQueryDuration(operation, table string, duration float64) {
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

// SetActiveConnections sets the number of active connections
func (m *Metrics) SetActiveConnections(count int) {
	m.activeConnections.Set(float64(count))
}

// RecordWalletOperation records a wallet operation
func (m *Metrics) RecordWalletOperation(operation, status string) {
	m.walletOperations.WithLabelValues(operation, status).Inc()
}

// SetWalletBalance sets the wallet balance for a user
func (m *Metrics) SetWalletBalance(userID, currency string, balance float64) {
	m.walletBalanceTotal.WithLabelValues(userID, currency).Set(balance)
}

// SetupMetricsEndpoint sets up a metrics endpoint for Prometheus
func SetupMetricsEndpoint(app *fiber.App, metrics *Metrics) {
	// Create a new HTTP handler for prometheus metrics
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})

	// Create an adapter for using the HTTP handler with Fiber
	app.Get("/metrics", func(c *fiber.Ctx) error {
		// Use fasthttpadaptor to convert the fiber context to an http handler
		handler := fasthttpadaptor.NewFastHTTPHandler(handler)
		handler(c.Context())
		return nil
	})
}

// Shutdown performs any necessary cleanup of metrics resources
func (m *Metrics) Shutdown(ctx context.Context) error {
	// Currently, Prometheus metrics don't require special shutdown handling
	// This method is provided for consistency with other observability components
	// and to allow for future extensions
	return nil
}
