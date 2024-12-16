package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mattmattox/kubebackup/pkg/backup"
	"github.com/mattmattox/kubebackup/pkg/config"
	"github.com/mattmattox/kubebackup/pkg/k8s"
	"github.com/mattmattox/kubebackup/pkg/logging"
	"github.com/mattmattox/kubebackup/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	logger         = logging.SetupLogging()
	taskLock       sync.Mutex
	isTaskRunning  bool
	lastBackupInfo = struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Time    string `json:"time"`
	}{
		Status:  "unknown",
		Message: "No backups have been run yet.",
		Time:    "",
	}

	// Prometheus Metrics
	lastBackupStatus   = prometheus.NewGauge(prometheus.GaugeOpts{Name: "last_backup_status", Help: "The status of the last backup: 1 for success, 0 for failure."})
	lastBackupTime     = prometheus.NewGauge(prometheus.GaugeOpts{Name: "last_backup_timestamp", Help: "Last successful backup timestamp."})
	lastBackupDuration = prometheus.NewGauge(prometheus.GaugeOpts{Name: "last_backup_duration_seconds", Help: "Duration of the last backup in seconds."})
)

func init() {
	// Register Prometheus Metrics
	prometheus.MustRegister(lastBackupStatus, lastBackupTime, lastBackupDuration)
}

func main() {
	flag.Parse() // Parse command-line flags

	// Load configuration from environment variables
	config.LoadConfiguration()

	logging.SetupLogging()

	// Validate configuration
	if err := validateConfig(); err != nil {
		logger.Fatalf("Configuration validation failed: %v", err)
	}

	// Connect to the Kubernetes cluster
	clientset, dynamicClient, err := k8s.ConnectToCluster(config.CFG.Kubeconfig)
	if err != nil {
		logger.Fatalf("Error creating clientset: %v", err)
	}

	// Verify access to the cluster
	err = k8s.VerifyAccessToCluster(clientset)
	if err != nil {
		logger.Fatalf("Error verifying access to cluster: %v", err)
	}

	// Start HTTP server for admin and metrics
	logger.Println("Starting HTTP server for metrics and admin endpoints...")
	server := startHTTPServer(clientset, dynamicClient)

	// Context to handle shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Handle RunOnce flag
	if config.CFG.RunOnce {
		logger.Println("RunOnce flag is enabled. Performing a single backup and exiting.")
		performBackup(clientset, dynamicClient)
		logger.Println("Task execution completed. Exiting...")
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Printf("Error during server shutdown: %v", err)
		}
		return
	}

	// Handle DisableCron flag
	if config.CFG.DisableCron {
		logger.Println("Cron scheduling is disabled. Exiting.")
		return
	}

	// Create and start a cron scheduler
	c := cron.New()
	_, err = c.AddFunc(config.CFG.CronSchedule, func() {
		logger.Println("Starting scheduled backup...")
		performBackup(clientset, dynamicClient)
	})
	if err != nil {
		logger.Fatalf("Error adding cron job: %v", err)
	}

	logger.Println("Starting cron scheduler...")
	c.Start()

	// Wait for termination signals
	go func() {
		<-signalChan
		logger.Println("Received shutdown signal, stopping cron scheduler...")
		c.Stop()
		cancel()
	}()

	// Wait for the context to be canceled
	<-ctx.Done()

	logger.Println("Exiting gracefully.")
}

// validateConfig ensures required fields are set in the configuration.
func validateConfig() error {
	if config.CFG.CronSchedule == "" {
		return fmt.Errorf("CronSchedule cannot be empty")
	}
	if config.CFG.BackupTarget == "s3" {
		if config.CFG.S3AccessKeyID == "" || config.CFG.S3SecretAccessKey == "" {
			return fmt.Errorf("S3 configuration is incomplete: missing AccessKeyID or SecretAccessKey")
		}
		if config.CFG.S3Bucket == "" {
			return fmt.Errorf("S3 configuration is incomplete: missing Bucket")
		}
	}
	return nil
}

// performBackup triggers the backup process and updates metrics/status.
func performBackup(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) {
	startTime := time.Now()

	status, err := backup.StartBackup(clientset, dynamicClient, &config.CFG)
	duration := time.Since(startTime)

	if err != nil {
		logger.Printf("Backup failed: %v", err)
		lastBackupInfo = struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Time    string `json:"time"`
		}{
			Status:  "failed",
			Message: err.Error(),
			Time:    startTime.Format(time.RFC3339),
		}
		lastBackupStatus.Set(0)
		return
	}

	if status {
		logger.Printf("Backup completed successfully in %v", duration)
		lastBackupInfo = struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Time    string `json:"time"`
		}{
			Status:  "success",
			Message: "Backup completed successfully.",
			Time:    startTime.Format(time.RFC3339),
		}
		lastBackupStatus.Set(1)
		lastBackupTime.Set(float64(startTime.Unix()))
		lastBackupDuration.Set(duration.Seconds())
	} else {
		logger.Printf("Backup completed with errors in %v", duration)
		lastBackupInfo = struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Time    string `json:"time"`
		}{
			Status:  "failed",
			Message: "Backup completed with errors.",
			Time:    startTime.Format(time.RFC3339),
		}
		lastBackupStatus.Set(0)
	}
}

// startHTTPServer starts an HTTP server for metrics and admin endpoints
func startHTTPServer(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) *http.Server {
	logger.Println("Setting up HTTP server...")
	mux := http.NewServeMux()

	mux.HandleFunc("/", defaultPage)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", healthCheck)
	mux.HandleFunc("/version", versionInfo)
	mux.HandleFunc("/backup", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("HTTP request to /backup from %s", r.RemoteAddr)
		if triggerAPITask(w, clientset, dynamicClient, "backup") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Backup triggered successfully at %s.\n", time.Now().Format(time.RFC3339))
		}
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("HTTP request to /status from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lastBackupInfo)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.CFG.MetricsPort),
		Handler:      logRequestMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Printf("HTTP server running on port %d", config.CFG.MetricsPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server failed: %v", err)
		}
	}()
	return server
}

// defaultPage returns a simple HTML page with links to various endpoints
func defaultPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
		<!DOCTYPE html>
		<html>
		<head><title>KubeBackup</title></head>
		<body>
			<h1>KubeBackup</h1>
			<ul>
				<li><a href="/metrics">Metrics</a></li>
				<li><a href="/healthz">Health Check</a></li>
				<li><a href="/version">Version</a></li>
				<li><a href="/backup">Trigger Backup</a></li>
			</ul>
		</body>
		</html>
	`)
}

// healthCheck returns a simple health check response
func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

// versionInfo returns the version information of the application
func versionInfo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		"version":   version.Version,
		"gitCommit": version.GitCommit,
		"buildTime": version.BuildTime,
	})
	if err != nil {
		logger.Printf("Failed to encode version info: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// logRequestMiddleware logs incoming HTTP requests
func logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Incoming request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// triggerAPITask triggers a task based on the mode and returns true if the task was started
func triggerAPITask(w http.ResponseWriter, clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, mode string) bool {
	taskLock.Lock()
	defer taskLock.Unlock()

	if isTaskRunning {
		logger.Printf("Task already running; skipping %s request.", mode)
		http.Error(w, "Another task is already running", http.StatusConflict)
		return false
	}

	isTaskRunning = true
	go func() {
		defer func() {
			taskLock.Lock()
			isTaskRunning = false
			taskLock.Unlock()
			logger.Printf("Task for mode %s completed.", mode)
		}()

		startTime := time.Now()
		logger.Printf("Starting %s task...", mode)

		switch mode {
		case "backup":
			performBackup(clientset, dynamicClient)
		default:
			logger.Printf("Invalid task mode: %s", mode)
			http.Error(w, "Invalid task mode", http.StatusBadRequest)
		}

		duration := time.Since(startTime)
		logger.Printf("%s task completed in %v", mode, duration)
	}()

	return true
}
