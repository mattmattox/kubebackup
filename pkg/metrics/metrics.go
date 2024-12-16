package metrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	backupDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kubebackup_backup_duration_seconds",
		Help: "Duration of the backup process in seconds.",
	})

	timeSinceLastBackup = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kubebackup_time_since_last_backup_seconds",
		Help: "Time since the last successful backup in seconds.",
	})

	backupSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kubebackup_backup_success",
		Help: "Indicates whether the last backup was successful (1) or not (0).",
	})

	objectCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kubebackup_objects_count",
		Help: "Number of objects backed up per object type.",
	}, []string{"object_type"})

	namespacesTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kubebackup_namespaces_total",
		Help: "Total number of namespaces being backed up.",
	})
)

func init() {
	prometheus.MustRegister(backupDuration)
	prometheus.MustRegister(timeSinceLastBackup)
	prometheus.MustRegister(backupSuccess)
	prometheus.MustRegister(objectCount)
	prometheus.MustRegister(namespacesTotal)
}

func StartMetricsServer(ctx context.Context, metricsPort string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: mux,
	}

	// Run the server in a goroutine
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background()) // Shutdown server on context cancellation
	}()

	return server.ListenAndServe()
}

func WriteBackupDuration(duration float64) {
	backupDuration.Set(duration)
}

func WriteTimeSinceLastBackup(duration float64) {
	timeSinceLastBackup.Set(duration)
}

func WriteBackupSuccess(success bool) {
	if success {
		backupSuccess.Set(1)
	} else {
		backupSuccess.Set(0)
	}
}

func WriteObjectCount(objectType string, count int) {
	objectCount.WithLabelValues(objectType).Add(float64(count))
}

func WriteNamespaceCount(count int) {
	namespacesTotal.Set(float64(count))
}
