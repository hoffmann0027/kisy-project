package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Application-level instrumentation beyond the HTTP layer: live WebSocket
// connections, background-worker activity and Postgres pool saturation. These
// feed the O1 alert rules (WS load, worker health, pool exhaustion). All
// series are low-cardinality so they are safe to expose on /metrics.

var (
	// wsActiveConnections counts WebSocket clients currently held by THIS
	// instance's hub. It is a per-instance gauge; sum across instances for a
	// fleet-wide total.
	wsActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kisy_ws_active_connections",
		Help: "WebSocket connections currently held by this instance.",
	})

	// workerRuns counts background-worker passes (one per ticker tick that ran
	// to completion or errored), workerItems the units of work processed, and
	// workerErrors the failed passes. Labelled by worker name.
	workerRuns = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kisy_worker_runs_total",
		Help: "Background-worker passes by worker.",
	}, []string{"worker"})
	workerItems = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kisy_worker_items_total",
		Help: "Units of work processed by background workers.",
	}, []string{"worker"})
	workerErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kisy_worker_errors_total",
		Help: "Failed background-worker passes by worker.",
	}, []string{"worker"})
)

// WSConnect / WSDisconnect track the live WebSocket connection count. They are
// called from the hub as clients register and unregister; calls must balance.
func WSConnect()    { wsActiveConnections.Inc() }
func WSDisconnect() { wsActiveConnections.Dec() }

// WorkerRun records one completed pass of the named worker.
func WorkerRun(worker string) { workerRuns.WithLabelValues(worker).Inc() }

// WorkerItems records n units of work processed by the named worker.
func WorkerItems(worker string, n int) {
	if n > 0 {
		workerItems.WithLabelValues(worker).Add(float64(n))
	}
}

// WorkerError records one failed pass of the named worker.
func WorkerError(worker string) { workerErrors.WithLabelValues(worker).Inc() }

// RegisterDBPool exposes pgxpool statistics as Prometheus gauges/counters,
// read live from the pool at scrape time (O3 alerts on saturation). Safe to
// call once at startup.
func RegisterDBPool(pool *pgxpool.Pool) {
	prometheus.MustRegister(&poolCollector{pool: pool})
}

type poolCollector struct {
	pool *pgxpool.Pool
}

var (
	poolTotalDesc = prometheus.NewDesc("kisy_db_pool_total_conns",
		"Total connections currently in the pool (idle + in-use).", nil, nil)
	poolAcquiredDesc = prometheus.NewDesc("kisy_db_pool_acquired_conns",
		"Connections currently checked out of the pool.", nil, nil)
	poolIdleDesc = prometheus.NewDesc("kisy_db_pool_idle_conns",
		"Idle connections in the pool.", nil, nil)
	poolMaxDesc = prometheus.NewDesc("kisy_db_pool_max_conns",
		"Configured maximum pool size.", nil, nil)
	poolConstructingDesc = prometheus.NewDesc("kisy_db_pool_constructing_conns",
		"Connections currently being established.", nil, nil)
	poolAcquireCountDesc = prometheus.NewDesc("kisy_db_pool_acquire_count_total",
		"Cumulative successful connection acquisitions.", nil, nil)
	poolEmptyAcquireDesc = prometheus.NewDesc("kisy_db_pool_empty_acquire_count_total",
		"Cumulative acquisitions that had to wait for an empty pool.", nil, nil)
	poolCanceledAcquireDesc = prometheus.NewDesc("kisy_db_pool_canceled_acquire_count_total",
		"Cumulative acquisitions cancelled before completing.", nil, nil)
)

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- poolTotalDesc
	ch <- poolAcquiredDesc
	ch <- poolIdleDesc
	ch <- poolMaxDesc
	ch <- poolConstructingDesc
	ch <- poolAcquireCountDesc
	ch <- poolEmptyAcquireDesc
	ch <- poolCanceledAcquireDesc
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(poolTotalDesc, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(poolAcquiredDesc, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(poolIdleDesc, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(poolMaxDesc, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(poolConstructingDesc, prometheus.GaugeValue, float64(s.ConstructingConns()))
	ch <- prometheus.MustNewConstMetric(poolAcquireCountDesc, prometheus.CounterValue, float64(s.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(poolEmptyAcquireDesc, prometheus.CounterValue, float64(s.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(poolCanceledAcquireDesc, prometheus.CounterValue, float64(s.CanceledAcquireCount()))
}
