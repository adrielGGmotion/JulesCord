package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CommandCounter tracks the number of times a command is executed.
	CommandCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "julescord_command_executions_total",
			Help: "Total number of times a command has been executed.",
		},
		[]string{"command"},
	)

	// CommandLatency tracks the execution time of commands.
	CommandLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "julescord_command_latency_seconds",
			Help:    "Latency of command executions in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command"},
	)

	// DBQueryLatency tracks the execution time of database queries.
	DBQueryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "julescord_db_query_latency_seconds",
			Help:    "Latency of database queries in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query"},
	)
)
