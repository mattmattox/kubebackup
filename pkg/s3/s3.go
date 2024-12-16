package s3

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mattmattox/kubebackup/pkg/logging"
)

var log = logging.SetupLogging()

func createS3Session(region, accessKey, secretKey, endpoint, caPath string, disableSSL bool) (*session.Session, error) {
	s3Config := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	}

	// Configure the endpoint and SSL settings
	if endpoint != "" {
		s3Config.Endpoint = aws.String(endpoint)
		s3Config.S3ForcePathStyle = aws.Bool(true)
		s3Config.DisableSSL = aws.Bool(disableSSL)
	}

	// Configure custom CA if provided
	if caPath != "" {
		caCertPool, err := loadCustomCACerts(caPath)
		if err != nil {
			return nil, fmt.Errorf("error loading custom CA certificates: %v", err)
		}
		s3Config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
	}

	// Create and return the session
	return session.NewSession(s3Config)
}

func loadCustomCACerts(caPath string) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()

	// Read the custom CA certificate
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("error reading CA certificate file: %v", err)
	}

	// Append the CA certificate to the cert pool
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	return caCertPool, nil
}

func uploadToS3(sess *session.Session, bucket, key, filename, s3Folder string) error {
	log.Infof("Uploading file: %s", filename)
	// Open the file for reading
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	uploader := s3manager.NewUploader(sess)

	s3Key := fmt.Sprintf("%s/%s", s3Folder, key)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	return nil
}

func cleanupOldBackups(sess *session.Session, s3Bucket, s3Folder string, retentionPeriod int) error {
	log.Infoln("Retrieving list of objects in S3 bucket...")

	log.Infoln("Retaining backups for", retentionPeriod, "days")

	svc := s3.New(sess)
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s3Bucket),
		Prefix: aws.String(fmt.Sprintf("%s/kubebackup-", s3Folder)),
	}

	objects, err := svc.ListObjectsV2(listObjectsInput)
	if err != nil {
		return fmt.Errorf("error listing objects in S3 bucket: %v", err)
	}

	threshold := time.Now().AddDate(0, 0, -retentionPeriod)
	for _, obj := range objects.Contents {
		if obj.LastModified.Before(threshold) {
			log.Infof("Deleting object %s from S3 bucket", *obj.Key)
			deleteObjectInput := &s3.DeleteObjectInput{
				Bucket: aws.String(s3Bucket),
				Key:    obj.Key,
			}

			_, err := svc.DeleteObject(deleteObjectInput)
			if err != nil {
				return fmt.Errorf("error deleting object from S3 bucket: %v", err)
			}
		}
	}

	return nil
}

func UploadToS3(tarFilePath, s3Bucket, s3Folder, s3Region, s3AccessKey, s3SecretKey, s3Endpoint, caPath string, disableSSL bool, retentionPeriod int) error {
	log.Infoln("Uploading tarball to S3...")

	// Create S3 session
	sess, err := createS3Session(s3Region, s3AccessKey, s3SecretKey, s3Endpoint, caPath, disableSSL)
	if err != nil {
		return fmt.Errorf("error creating S3 session: %v", err)
	}

	// Upload tarball
	timestamp := filepath.Base(tarFilePath) // Use the tarball name as the key
	s3Key := fmt.Sprintf("%s/%s", s3Folder, timestamp)
	if err := uploadToS3(sess, s3Bucket, s3Key, tarFilePath, s3Folder); err != nil {
		return fmt.Errorf("failed to upload tarball to S3: %v", err)
	}

	// Cleanup old backups
	if retentionPeriod > 0 {
		log.Infof("Cleaning up old backups older than %d days...", retentionPeriod)
		if err := cleanupOldBackups(sess, s3Bucket, s3Folder, retentionPeriod); err != nil {
			return fmt.Errorf("error cleaning up old backups: %v", err)
		}
	}

	log.Infof("Backup successfully uploaded to S3: %s/%s", s3Bucket, s3Key)
	return nil
}
