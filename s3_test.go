package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestCompressAndUploadToS3(t *testing.T) {
	// Set up test variables (Replace these values with your own S3 bucket and credentials)
	s3Bucket := "your-s3-bucket-name"
	s3Region := "your-s3-region"
	s3AccessKey := "your-s3-access-key"
	s3SecretKey := "your-s3-secret-key"
	s3Endpoint := "" // Leave empty for AWS, provide an endpoint for other S3-compatible services

	// Create a temporary directory and write a test file
	tmpDir, err := ioutil.TempDir("", "s3-test-")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := tmpDir + "/test.txt"
	err = ioutil.WriteFile(testFile, []byte("Test content"), 0644)
	if err != nil {
		t.Fatalf("Error writing test file: %v", err)
	}

	// Compress the temporary directory and upload it to S3
	compressAndUploadToS3(tmpDir, s3Bucket, s3Region, s3AccessKey, s3SecretKey, s3Endpoint)

	// Verify that the file was uploaded to S3
	sess, err := createS3Session(s3Region, s3AccessKey, s3SecretKey, s3Endpoint)
	if err != nil {
		t.Fatalf("Error creating S3 session: %v", err)
	}

	s3Client := s3.New(sess)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	s3Key := fmt.Sprintf("kubebackup-%s.tar.gz", timestamp)
	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3Key),
	}

	_, err = s3Client.HeadObject(headObjectInput)
	if err != nil {
		t.Fatalf("Error checking if file exists in S3 bucket: %v", err)
	}
}
