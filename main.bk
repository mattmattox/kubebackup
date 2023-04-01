package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"archive/tar"
	"compress/gzip"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Read environment variable or use the provided default value
func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}

func main() {
	// Command line flags
	kubeconfigPtr := flag.String("kubeconfig", "", "Path to kubeconfig file")
	outputDirPtr := flag.String("output-dir", ".", "Path to output directory")
	s3Bucket := flag.String("s3-bucket", getEnv("S3_BUCKET", "my-bucket"), "S3 bucket to store backups")
	s3Region := flag.String("s3-region", getEnv("S3_REGION", "us-east-1"), "S3 region")
	s3AccessKey := flag.String("s3-access-key", getEnv("S3_ACCESS_KEY", ""), "S3 access key")
	s3SecretKey := flag.String("s3-secret-key", getEnv("S3_SECRET_KEY", ""), "S3 secret key")
	s3Endpoint := flag.String("s3-endpoint", getEnv("S3_ENDPOINT", ""), "S3 endpoint URL")
	flag.Parse()

	// Load kubeconfig if provided
	config, err := loadKubeConfig(*kubeconfigPtr)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Create a temporary directory
	tmpOutputDir, err := ioutil.TempDir("", "kubebackup-*")
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpOutputDir)

	// Process namespaced objects
	objects := getNamespacedObjects(clientset)
	namespaces, err := getNamespaces(clientset)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	for _, namespace := range namespaces {
		processNamespace(clientset, config, objects, namespace, *outputDirPtr)
	}

	// Process cluster-scoped objects
	processClusterScopedObjects(clientset, config, *outputDirPtr)

	// Compress the temporary directory and upload to S3
	compressAndUploadToS3(tmpOutputDir, *s3Bucket, *s3Region, *s3AccessKey, *s3SecretKey, *s3Endpoint)
}

func loadKubeConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func getNamespacedObjects(clientset *kubernetes.Clientset) []schema.GroupVersionResource {
	apiResourceList, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	objects := make([]schema.GroupVersionResource, 0)
	for _, apiResources := range apiResourceList {
		gv, _ := schema.ParseGroupVersion(apiResources.GroupVersion)
		for _, apiResource := range apiResources.APIResources {
			if apiResource.Namespaced {
				objects = append(objects, gv.WithResource(apiResource.Name))
			}
		}
	}
	return objects
}

func getNamespaces(clientset *kubernetes.Clientset) ([]string, error) {
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	namespaces := make([]string, len(namespaceList.Items))
	for i, namespace := range namespaceList.Items {
		namespaces[i] = namespace.GetName()
	}
	return namespaces, nil
}

func processNamespace(clientset *kubernetes.Clientset, config *rest.Config, objects []schema.GroupVersionResource, namespace string, outputDir string) {
	namespaceDir := filepath.Join(outputDir, "namespace", namespace)
	os.MkdirAll(namespaceDir, 0755)
	fmt.Printf("Namespace: %s\n", namespace)

	dynamicClient := dynamic.NewForConfigOrDie(config)

	for _, object := range objects {
		objectDir := filepath.Join(namespaceDir, object.Resource)
		os.MkdirAll(objectDir, 0755)
		fmt.Printf("Object: %s\n", object.Resource)

		unstructuredList, err := dynamicClient.Resource(object).Namespace(namespace).List(context.Background(), v1.ListOptions{})
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		for _, item := range unstructuredList.Items {
			itemBytes, err := item.MarshalJSON()
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			itemYaml, err := yaml.JSONToYAML(itemBytes)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			itemFile := filepath.Join(objectDir, fmt.Sprintf("%s.yaml", item.GetName()))
			err = ioutil.WriteFile(itemFile, itemYaml, 0644)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
		}
	}
}

func processClusterScopedObjects(clientset *kubernetes.Clientset, config *rest.Config, outputDir string) {
	clusterDir := filepath.Join(outputDir, "clusterobjects")
	os.MkdirAll(clusterDir, 0755)

	apiResourceList, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	dynamicClient := dynamic.NewForConfigOrDie(config)

	for _, apiResources := range apiResourceList {
		gv, _ := schema.ParseGroupVersion(apiResources.GroupVersion)
		for _, apiResource := range apiResources.APIResources {
			if !apiResource.Namespaced {
				clusterObjectDir := filepath.Join(clusterDir, apiResource.Name)
				os.MkdirAll(clusterObjectDir, 0755)
				fmt.Printf("Cluster Object: %s\n", apiResource.Name)

				object := gv.WithResource(apiResource.Name)
				unstructuredList, err := dynamicClient.Resource(object).List(context.Background(), v1.ListOptions{})
				if err != nil {
					fmt.Println("Error:", err)
					os.Exit(1)
				}
				for _, item := range unstructuredList.Items {
					itemBytes, err := item.MarshalJSON()
					if err != nil {
						fmt.Println("Error:", err)
						os.Exit(1)
					}
					itemYaml, err := yaml.JSONToYAML(itemBytes)
					if err != nil {
						fmt.Println("Error:", err)
						os.Exit(1)
					}
					itemFile := filepath.Join(clusterObjectDir, fmt.Sprintf("%s.yaml", item.GetName()))
					err = ioutil.WriteFile(itemFile, itemYaml, 0644)
					if err != nil {
						fmt.Println("Error:", err)
						os.Exit(1)
					}
				}
			}
		}
	}
}

func compressAndUploadToS3(src string, s3Bucket, s3Region, s3AccessKey, s3SecretKey, s3Endpoint string) {
	timestamp := time.Now().Format("20060102150405")
	tarGzFilename := fmt.Sprintf("kubebackup-%s.tar.gz", timestamp)

	// Create a .tar.gz file
	tarGzFile, err := os.Create(tarGzFilename)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer tarGzFile.Close()

	// Create a gzip.Writer and tar.Writer
	gzWriter := gzip.NewWriter(tarGzFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Compress the source directory
	err = filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(file, src)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
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

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Upload the .tar.gz file to S3
	uploadToS3(tarGzFilename, s3Bucket, s3Region, s3AccessKey, s3SecretKey, s3Endpoint)
}

func uploadToS3(filename, s3Bucket, s3Region, s3AccessKey, s3SecretKey, s3Endpoint string) {
	// Initialize an S3 session
	s3Config := &aws.Config{
		Region:      aws.String(s3Region),
		Credentials: credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, ""),
	}

	if s3Endpoint != "" {
		s3Config.Endpoint = aws.String(s3Endpoint)
		s3Config.DisableSSL = aws.Bool(true) // Disable SSL if you are using a self-signed certificate or an insecure connection
		s3Config.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(s3Config)

	// Open the file for reading
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer file.Close()

	// Upload the tar.gz file to S3
	err = uploadToS3WithSession(sess, s3Bucket, filename)
	if err != nil {
		fmt.Println("Error uploading to S3:", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully uploaded %s to %s\n", filename, s3Bucket)
}

func uploadToS3WithSession(sess *session.Session, s3Bucket, filename string) error {
	// Use sess to create an uploader
	uploader := s3manager.NewUploader(sess)

	// Open the file for reading
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Upload the file to S3
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(filepath.Base(filename)),
		Body:   file,
	})

	return err
}
