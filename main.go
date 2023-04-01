package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var interval int

var (
	backupDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "kubebackup_duration_seconds",
		Help: "Duration of the backup process in seconds",
	})

	backupSuccess = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kubebackup_success",
		Help: "Indicates if the last backup was successful (1) or not (0)",
	})

	lastBackupTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kubebackup_last_backup_timestamp_seconds",
		Help: "Timestamp of the last successful backup",
	})

	objectCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kubebackup_objects_count",
		Help: "Number of objects backed up by object type",
	}, []string{"type"})
)

func serveMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9009", nil)
}

func init() {
	flag.IntVar(&interval, "interval", 0, "Interval in hours between backups. Default is 12 hours.")
}

func main() {
	kubeconfigPtr := flag.String("kubeconfig", "", "Path to kubeconfig file")
	s3Bucket := flag.String("s3-bucket", getEnv("S3_BUCKET", "my-bucket"), "S3 bucket to store backups")
	s3Region := flag.String("s3-region", getEnv("S3_REGION", "us-east-1"), "S3 region")
	s3AccessKey := flag.String("s3-access-key", getEnv("S3_ACCESS_KEY", ""), "S3 access key")
	s3SecretKey := flag.String("s3-secret-key", getEnv("S3_SECRET_KEY", ""), "S3 secret key")
	s3Endpoint := flag.String("s3-endpoint", getEnv("S3_ENDPOINT", ""), "S3 endpoint URL")

	flag.Parse()

	if interval == 0 {
		intervalEnv := os.Getenv("INTERVAL")
		if intervalEnv != "" {
			var err error
			interval, err = strconv.Atoi(intervalEnv)
			if err != nil {
				log.Fatalf("Error parsing INTERVAL environment variable: %v", err)
			}
		} else {
			interval = 12
		}
	}
	// Start the metrics server
	go serveMetrics()

	for {
		tmpDir, err := ioutil.TempDir("", "kubebackup-")
		if err != nil {
			log.Fatalf("Error creating temporary directory: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		backup(*kubeconfigPtr, tmpDir, *s3Bucket, *s3Region, *s3AccessKey, *s3SecretKey, *s3Endpoint)

		log.Printf("Waiting for %d hours before the next backup...\n", interval)
		time.Sleep(time.Duration(interval) * time.Hour)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func backup(kubeconfig, tmpDir, s3Bucket, s3Region, s3AccessKey, s3SecretKey, s3Endpoint string) {
	startTime := time.Now()
	success := false
	defer func() {
		backupDuration.Observe(time.Since(startTime).Seconds())
		if success {
			backupSuccess.Set(1)
			lastBackupTimestamp.SetToCurrentTime()
		} else {
			backupSuccess.Set(0)
		}
	}()

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error getting in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}

	// Get namespace and namespaced resources
	namespacedResources, err := getNamespacedObjects(clientset)
	if err != nil {
		log.Fatalf("Error fetching namespaced resources: %v", err)
	}

	namespaces, err := getNamespaces(clientset)
	if err != nil {
		log.Fatalf("Error fetching namespaces: %v", err)
	}

	for _, ns := range namespaces {
		processNamespace(clientset, config, namespacedResources, ns, tmpDir)
	}

	processClusterScopedObjects(clientset, config, tmpDir)

	compressAndUploadToS3(tmpDir, s3Bucket, s3Region, s3AccessKey, s3SecretKey, s3Endpoint)
}
