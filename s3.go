package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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

func compressAndUploadToS3(tmpDir, s3Bucket, s3Folder, s3Region, s3AccessKey, s3SecretKey, s3Endpoint string) error {
	log.Infoln("Compressing and uploading backup to S3...")
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("kubebackup_%s.tar.gz", timestamp)

	// Create tarball
	log.Infoln("Creating tarball...")
	tarFilePath := filepath.Join(tmpDir, filename)
	err := createTarball(tmpDir, tarFilePath)
	if err != nil {
		return fmt.Errorf("error creating tarball: %v", err)
	}
	defer os.Remove(tarFilePath)

	// Upload to S3
	log.Infoln("Uploading tarball to S3...")
	sess, err := createS3Session(s3Region, s3AccessKey, s3SecretKey, s3Endpoint)
	if err != nil {
		return fmt.Errorf("error creating S3 session: %v", err)
	}

	s3Key := fmt.Sprintf("kubebackup-%s.tar.gz", timestamp)
	err = uploadToS3(sess, s3Bucket, s3Key, tarFilePath, s3Folder)
	if err != nil {
		return fmt.Errorf("failed to upload tarball to S3: %v", err)
	}

	log.Infoln("Cleaning up old backups...")
	err = cleanupOldBackups(sess, s3Bucket, s3Folder, retentionPeriod)
	if err != nil {
		return fmt.Errorf("error cleaning up old backups: %v", err)
	}

	return nil
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
