package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func createS3Session(region, accessKey, secretKey, endpoint string) (*session.Session, error) {
	s3Config := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	}

	if endpoint != "" {
		s3Config.Endpoint = aws.String(endpoint)
		s3Config.DisableSSL = aws.Bool(true)
		s3Config.S3ForcePathStyle = aws.Bool(true)
	}

	return session.NewSession(s3Config)
}

func compressAndUploadToS3(tmpDir, s3Bucket, s3Region, s3AccessKey, s3SecretKey, s3Endpoint string) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("kubebackup_%s.tar.gz", timestamp)

	// Create tarball
	tarFilePath := filepath.Join(tmpDir, filename)
	err := createTarball(tmpDir, tarFilePath)
	if err != nil {
		fmt.Printf("Error creating tarball: %v\n", err)
		return
	}
	defer os.Remove(tarFilePath)

	// Upload to S3
	sess, err := createS3Session(s3Region, s3AccessKey, s3SecretKey, s3Endpoint)
	if err != nil {
		fmt.Printf("Error creating S3 session: %v\n", err)
		return
	}

	s3Key := fmt.Sprintf("kubebackup-%s.tar.gz", timestamp)
	err = uploadToS3(sess, s3Bucket, s3Key, tarFilePath)
	if err != nil {
		log.Fatalf("Failed to upload tarball to S3: %v", err)
	}
}

func createTarball(src, tarFilePath string) error {
	tarFile, err := os.Create(tarFilePath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(src, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = filepath.Join(filepath.Base(src), file[len(src):])

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tarWriter, f)
		return err
	})
}

func uploadToS3(sess *session.Session, bucket, key, filename string) error {
	// Open the file for reading
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	return nil
}
