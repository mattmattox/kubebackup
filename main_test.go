package main

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMainBackup(t *testing.T) {
	// Create a fake Kubernetes client for testing
	clientset := fake.NewSimpleClientset()

	// Add a test namespace to the fake clientset
	testNamespace := &v1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), testNamespace, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test namespace: %v", err)
	}

	// Create a temporary output directory
	tmpDir, err := ioutil.TempDir("", "kube-test-")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up a local MinIO server for testing
	endpoint := "play.min.io" // Use MinIO's public play server for testing
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"

	// Create MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		t.Fatalf("Error creating MinIO client: %v", err)
	}

	// Create a test S3 bucket
	s3Bucket := "test-kube-backup"
	err = minioClient.MakeBucket(context.Background(), s3Bucket, minio.MakeBucketOptions{})
	if err != nil {
		t.Fatalf("Error creating test S3 bucket: %v", err)
	}
	defer minioClient.RemoveBucket(context.Background(), s3Bucket)

	// Run the backup function
	backup(clientset, "", tmpDir, s3Bucket, "", accessKeyID, secretAccessKey, endpoint)

	// Check if the backup file was uploaded to the S3 bucket
	objectName := "kubebackup-"
	doneCh := make(chan struct{})
	defer close(doneCh)
	found := false
	for object := range minioClient.ListObjects(context.Background(), s3Bucket, minio.ListObjectsOptions{Prefix: objectName, Recursive: false},
