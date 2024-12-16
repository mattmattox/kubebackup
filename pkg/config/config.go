package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// AppConfig structure for environment-based configurations.
type AppConfig struct {
	Debug             bool   `json:"debug"`
	LogLevel          string `json:"log_level"`
	MetricsPort       int    `json:"metricsPort"`
	Kubeconfig        string `json:"kubeconfig"`
	BackupDir         string `json:"backup_dir"`
	CronSchedule      string `json:"cron_schedule"`
	DisableCron       bool   `json:"disable_cron"`
	RunOnce           bool   `json:"run_once"`
	Retention         int    `json:"retention"`
	BackupTarget      string `json:"backup_target"`
	S3Endpoint        string `json:"s3Endpoint"`
	S3AccessKeyID     string `json:"s3AccessKeyID"`
	S3SecretAccessKey string `json:"s3SecretAccessKey"`
	S3Bucket          string `json:"s3Bucket"`
	S3Region          string `json:"s3Region"`
	S3Folder          string `json:"s3_folder"`
	S3DisableSSL      bool   `json:"s3_disable_ssl"`
	S3CustomCA        string `json:"s3_custom_ca"`
	S3CustomCAPath    string `json:"s3_custom_ca_path"`
}

// CFG is the global configuration object.
var CFG AppConfig

// LoadConfiguration loads configuration from environment variables.
func LoadConfiguration() {
	CFG.Debug = parseEnvBool("DEBUG", false)
	CFG.MetricsPort = parseEnvInt("METRICS_PORT", 9999)
	CFG.S3Endpoint = getEnvOrDefault("S3_ENDPOINT", "s3.us-central-1.wasabisys.com")
	CFG.S3AccessKeyID = getEnvOrDefault("S3_ACCESS_KEY_ID", "")
	CFG.S3SecretAccessKey = getEnvOrDefault("S3_SECRET_ACCESS_KEY", "")
	CFG.S3Bucket = getEnvOrDefault("S3_BUCKET", "")
	CFG.S3Region = getEnvOrDefault("S3_REGION", "")
	CFG.S3Folder = getEnvOrDefault("S3_FOLDER", "")
	CFG.S3DisableSSL = parseEnvBool("S3_DISABLE_SSL", false)
	CFG.S3CustomCA = getEnvOrDefault("S3_CUSTOM_CA", "")
	CFG.S3CustomCAPath = getEnvOrDefault("S3_CUSTOM_CA_PATH", "")
	CFG.BackupDir = getEnvOrDefault("BACKUP_DIR", "/pvc")
	CFG.Retention = parseEnvInt("RETENTION", 30)
	CFG.CronSchedule = getEnvOrDefault("CRON_SCHEDULE", "0 0 * * *")
	CFG.DisableCron = parseEnvBool("DISABLE_CRON", false)
	CFG.RunOnce = parseEnvBool("RUN_ONCE", false)
	CFG.BackupTarget = getEnvOrDefault("BACKUP_TARGET", "s3")
	CFG.Kubeconfig = getEnvOrDefault("KUBECONFIG", "~/.kube/config")
	CFG.LogLevel = getEnvOrDefault("LOG_LEVEL", "info")
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func parseEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var intValue int
	_, err := fmt.Sscanf(value, "%d", &intValue)
	if err != nil {
		log.Printf("Failed to parse environment variable %s: %v. Using default value: %d", key, err, defaultValue)
		return defaultValue
	}
	return intValue
}

func parseEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value = strings.ToLower(value)

	// Handle additional truthy and falsy values
	switch value {
	case "1", "t", "true", "yes", "on", "enabled":
		return true
	case "0", "f", "false", "no", "off", "disabled":
		return false
	default:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Error parsing %s as bool: %v. Using default value: %t", key, err, defaultValue)
			return defaultValue
		}
		return boolValue
	}
}
