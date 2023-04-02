package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

var (
	interval        int
	retentionPeriod int
	metricsport     int
	log             = logrus.New()
	kubeconfig      string

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
	flag.IntVar(&interval, "interval", 0, "Interval in hours between backups. Default is 12 hours.")

	// Register Prometheus metrics
	prometheus.MustRegister(backupDuration)
	prometheus.MustRegister(timeSinceLastBackup)
	prometheus.MustRegister(backupSuccess)
	prometheus.MustRegister(objectCount)
	prometheus.MustRegister(namespacesTotal)
}

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file. (Optional, defaults to in-cluster config)")
	s3Bucket := flag.String("s3-bucket", getEnv("S3_BUCKET", "my-bucket"), "S3 bucket to store backups")
	s3Folder := flag.String("s3-folder", getEnv("S3_FOLDER", "my-cluster"), "S3 folder to store backups")
	s3Region := flag.String("s3-region", getEnv("S3_REGION", "us-east-1"), "S3 region")
	s3Endpoint := flag.String("s3-endpoint", getEnv("S3_ENDPOINT", ""), "S3 endpoint URL")
	s3AccessKey := flag.String("s3-access-key", getEnv("S3_ACCESS_KEY", ""), "S3 access key")
	s3SecretKey := flag.String("s3-secret-key", getEnv("S3_SECRET_KEY", ""), "S3 secret key")

	flag.Parse()

	if metricsport == 0 {
		metricsportEnv := os.Getenv("METRICSPORT")
		if metricsportEnv != "" {
			var err error
			metricsport, err = strconv.Atoi(metricsportEnv)
			if err != nil {
				log.Fatalf("Error parsing METRICSPORT environment variable: %v", err)
			}
		} else {
			metricsport = 9009
		}
	}

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

	if retentionPeriod == 0 {
		retentionPeriodEnv := os.Getenv("RETENTION_PERIOD")
		if retentionPeriodEnv != "" {
			var err error
			retentionPeriod, err = strconv.Atoi(retentionPeriodEnv)
			if err != nil {
				log.Fatalf("Error parsing RETENTION_PERIOD environment variable: %v", err)
			}
		} else {
			retentionPeriod = 30
		}
	}

	logLevel := getEnv("LOG_LEVEL", "info")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(level)

	go func() {
		log.Debugln("Starting metrics server on port %s", metricsport)
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/", http.RedirectHandler("/metrics", http.StatusMovedPermanently))
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		http.ListenAndServe(":"+strconv.Itoa(metricsport), nil)
	}()

	for {
		log.Infoln("Starting backup process...")

		log.Infoln("Creating temporary directory for backup...")
		tmpDir, err := ioutil.TempDir("", "kubebackup-")
		if err != nil {
			log.Fatalf("Error creating temporary directory: %v", err)
		}
		log.Infof("Temporary directory created successfully at path: %s\n", tmpDir)

		startTime := time.Now()
		success := backup(kubeconfig, tmpDir, *s3Bucket, *s3Folder, *s3Region, *s3AccessKey, *s3SecretKey, *s3Endpoint)
		duration := time.Since(startTime).Seconds()

		backupDuration.Set(duration)
		timeSinceLastBackup.Set(0)
		if success {
			backupSuccess.Set(1)
			log.Infoln("Backup completed successfully!")
		} else {
			backupSuccess.Set(0)
			cleanupTmpDir(tmpDir)
			log.Fatalf("Backup failed!")
		}

		log.Infof("Backup duration: %f seconds", duration)

		log.Infoln("Cleaning up temporary directory...")
		cleanupTmpDir(tmpDir)

		log.Infof("Waiting for %d hours before the next backup...\n", interval)
		time.Sleep(time.Duration(interval) * time.Hour)

		// Update time since last backup metric
		timeSinceLastBackup.Inc()
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func backup(kubeconfig, tmpDir, s3Bucket, s3Folder, s3Region, s3AccessKey, s3SecretKey, s3Endpoint string) bool {
	log.Infoln("Loading Kubernetes config...")
	config, err := loadKubeConfig(kubeconfig)
	if err != nil {
		log.Printf("Error loading kubeconfig: %v", err)
		return false
	}

	log.Infoln("Creating Kubernetes clientset...")
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Error creating clientset: %v", err)
		return false
	}

	log.Infoln("Fetching namespaced resources...")
	namespacedResources, err := getNamespacedObjects(clientset)
	if err != nil {
		log.Printf("Error fetching namespaced resources: %v", err)
		return false
	}

	log.Infoln("Fetching namespaces...")
	namespaces, err := getNamespaces(clientset)
	if err != nil {
		log.Printf("Error fetching namespaces: %v", err)
		return false
	}

	log.Infof("Found %d namespaces", len(namespaces))

	// Update the number of namespaces metric
	namespacesTotal.Set(float64(len(namespaces)))

	for _, ns := range namespaces {
		processNamespace(clientset, config, namespacedResources, ns, tmpDir)
	}

	processClusterScopedObjects(clientset, config, tmpDir)

	err = compressAndUploadToS3(tmpDir, s3Bucket, s3Folder, s3Region, s3AccessKey, s3SecretKey, s3Endpoint)
	if err != nil {
		log.Printf("Error compressing and uploading to S3: %v", err)
		return false
	}

	sess, err := createS3Session(s3Region, s3AccessKey, s3SecretKey, s3Endpoint)
	if err != nil {
		log.Printf("Error creating S3 session: %v", err)
		return false
	}

	err = cleanupOldBackups(sess, s3Bucket, s3Folder, retentionPeriod)
	if err != nil {
		log.Printf("Error deleting old backups: %v", err)
		return false
	}

	return true
}

func cleanupTmpDir(tmpDir string) {
	log.Infof("Cleaning up temporary directory %s...", tmpDir)
	if err := os.RemoveAll(tmpDir); err != nil {
		log.Errorf("Error cleaning up temporary directory %s: %v", tmpDir, err)
	} else {
		log.Infof("Temporary directory %s cleaned up successfully", tmpDir)
	}
}
