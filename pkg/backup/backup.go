package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mattmattox/kubebackup/pkg/config"
	"github.com/mattmattox/kubebackup/pkg/k8s"
	"github.com/mattmattox/kubebackup/pkg/logging"
	"github.com/mattmattox/kubebackup/pkg/s3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var log = logging.SetupLogging()

func StartBackup(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, cfg *config.AppConfig) (bool, error) {
	log.Infoln("Fetching namespaces...")
	namespaces, err := k8s.GetNamespaces(clientset)
	if err != nil {
		return false, fmt.Errorf("error fetching namespaces: %v", err)
	}
	log.Infof("Found %d namespaces.", len(namespaces))

	// Create a temporary directory for storing backup files
	tmpDir, err := os.MkdirTemp("", "kubebackup-")
	if err != nil {
		return false, fmt.Errorf("error creating temporary directory: %v", err)
	}
	log.Infof("Created temporary directory: %s", tmpDir)

	// Ensure the temporary directory is cleaned up after the tar operation
	defer func() {
		log.Infof("Cleaning up temporary directory: %s", tmpDir)
		os.RemoveAll(tmpDir)
	}()

	// Process cluster-scoped resources
	log.Infoln("Fetching cluster-scoped resources...")
	clusterScopedResources, err := k8s.GetClusterScopedResources(clientset)
	if err != nil {
		return false, fmt.Errorf("error fetching cluster-scoped resources: %v", err)
	}
	log.Infof("Found %d cluster-scoped resources.", len(clusterScopedResources))

	clusterScopedDir := filepath.Join(tmpDir, "cluster-scoped")
	log.Infoln("Processing cluster-scoped resources...")
	if err := ProcessClusterScopedResources(dynamicClient, clusterScopedResources, clusterScopedDir); err != nil {
		return false, fmt.Errorf("error processing cluster-scoped resources: %v", err)
	}
	log.Infof("Cluster-scoped resources processed successfully.")

	// Process namespace-scoped resources
	log.Infoln("Fetching namespaced resources...")
	namespacedResources, err := k8s.GetNamespaceScopedResources(clientset)
	if err != nil {
		return false, fmt.Errorf("error fetching namespaced resources: %v", err)
	}
	log.Infof("Found %d namespaced resources.", len(namespacedResources))

	namespaceScopedDir := filepath.Join(tmpDir, "namespace-scoped")
	log.Infoln("Processing namespace-scoped resources...")
	if err := ProcessNamespaces(dynamicClient, namespaces, namespacedResources, namespaceScopedDir); err != nil {
		return false, fmt.Errorf("error processing namespace-scoped resources: %v", err)
	}
	log.Infof("Namespace-scoped resources processed successfully.")

	// Compress the backup directory
	tarFilePath, err := CompressBackup(tmpDir)
	if err != nil {
		return false, fmt.Errorf("error during compression: %v", err)
	}

	if cfg.BackupTarget == "s3" {
		// Compress and upload to S3
		log.Infoln("Uploading backup to S3...")
		if err := s3.UploadToS3(tarFilePath, cfg.S3Bucket, cfg.S3Folder, cfg.S3Region, cfg.S3AccessKeyID, cfg.S3SecretAccessKey, cfg.S3Endpoint, cfg.S3CustomCA, cfg.S3DisableSSL, cfg.Retention); err != nil {
			return false, fmt.Errorf("error compressing and uploading to S3: %v", err)
		}

		// Delete the tarball after uploading to S3
		if err := os.Remove(tarFilePath); err != nil {
			log.Errorf("Failed to delete tarball: %v", err)
		}
	} else {
		log.Infof("Backup tarball created at: %s", tarFilePath)
	}

	// Ensure cleanup of the temporary directory
	defer func() {
		if cleanupErr := CleanupTmpDir(tmpDir); cleanupErr != nil {
			log.Errorf("Failed to clean up temporary directory: %v", cleanupErr)
		}
	}()

	log.Infof("Backup process completed successfully.")
	return true, nil
}

func ProcessNamespaces(dynamicClient dynamic.Interface, namespaces []string, namespacedResources []schema.GroupVersionResource, baseDir string) error {
	var wg sync.WaitGroup
	var processErr error
	mu := &sync.Mutex{}

	for _, ns := range namespaces {
		wg.Add(1)
		go func(ns string) {
			defer wg.Done()
			if err := processNamespace(dynamicClient, ns, namespacedResources, baseDir); err != nil {
				mu.Lock()
				defer mu.Unlock()
				processErr = fmt.Errorf("error processing namespace '%s': %w", ns, err)
				log.Errorf("Error processing namespace '%s': %v", ns, err)
			}
		}(ns)
	}

	wg.Wait()

	return processErr
}

func ProcessClusterScopedResources(dynamicClient dynamic.Interface, resources []schema.GroupVersionResource, baseDir string) error {
	// Create the base directory for cluster-scoped resources
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("error creating directory '%s': %v", baseDir, err)
	}

	for _, resource := range resources {
		log.Infof("Processing cluster-scoped resource: %s", resource.Resource)

		// Fetch objects for the resource
		objects, err := GetClusterObjects(dynamicClient, resource)
		if err != nil {
			log.Errorf("Error fetching objects for resource '%s': %v", resource.Resource, err)
			continue
		}

		// Skip if no objects are found
		if len(objects) == 0 {
			log.Infof("No objects found for resource '%s'", resource.Resource)
			continue
		}

		// Create a directory for the resource
		resourceDir := filepath.Join(baseDir, resource.Resource)
		if err := os.MkdirAll(resourceDir, 0755); err != nil {
			log.Errorf("Error creating directory '%s': %v", resourceDir, err)
			continue
		}

		for _, object := range objects {
			objectName := object.GetName()
			objectFile := filepath.Join(resourceDir, objectName+".yaml")

			objectData, err := object.MarshalJSON()
			if err != nil {
				log.Errorf("Error marshalling object '%s': %v", objectName, err)
				continue
			}

			if err := writeObject(objectData, objectFile); err != nil {
				log.Errorf("Error writing object '%s' to file '%s': %v", objectName, objectFile, err)
				continue
			}
		}
	}

	return nil
}

func GetClusterObjects(dynamicClient dynamic.Interface, resource schema.GroupVersionResource) ([]*unstructured.Unstructured, error) {
	// Fetch the list of objects for the resource
	resourceList, err := dynamicClient.Resource(resource).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error fetching resource list for '%s': %v", resource.Resource, err)
	}

	// Convert items to []*unstructured.Unstructured
	objects := make([]*unstructured.Unstructured, 0, len(resourceList.Items))
	for i := range resourceList.Items {
		objects = append(objects, &resourceList.Items[i])
	}

	return objects, nil
}

func processNamespace(dynamicClient dynamic.Interface, ns string, namespacedResources []schema.GroupVersionResource, baseDir string) error {
	log.Infof("Processing namespace %s", ns)
	namespaceDir := fmt.Sprintf("%s/%s", baseDir, ns)
	if err := os.MkdirAll(namespaceDir, 0755); err != nil {
		return fmt.Errorf("error creating directory '%s': %v", namespaceDir, err)
	}

	for _, resource := range namespacedResources {
		if err := processResource(dynamicClient, ns, resource, namespaceDir); err != nil {
			log.Errorf("Error processing resource '%s' in namespace '%s': %v", resource.Resource, ns, err)
		}
	}
	return nil
}

func processResource(dynamicClient dynamic.Interface, ns string, resource schema.GroupVersionResource, namespaceDir string) error {
	log.Infof("Processing resource %s in namespace %s", resource.Resource, ns)
	objects, err := k8s.GetNamespaceObjects(dynamicClient, ns, resource, "")
	if err != nil {
		return fmt.Errorf("error fetching objects for resource '%s' in namespace '%s': %v", resource.Resource, ns, err)
	}

	log.Infof("Found %d objects for resource %s in namespace %s", len(objects), resource.Resource, ns)
	for _, object := range objects {
		if err := processObject(dynamicClient, ns, resource, object, namespaceDir); err != nil {
			log.Errorf("Error processing object '%s' of resource '%s': %v", object, resource.Resource, err)
		}
	}
	return nil
}

func processObject(dynamicClient dynamic.Interface, ns string, resource schema.GroupVersionResource, object string, namespaceDir string) error {
	log.Infof("Processing object %s of resource %s in namespace %s", object, resource.Resource, ns)

	// Create the directory for the resource
	objectDir := filepath.Join(namespaceDir, resource.Resource)
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		return fmt.Errorf("error creating directory '%s': %v", objectDir, err)
	}

	// Fetch the object data
	objectData, objectName, err := getObject(dynamicClient, ns, resource, object)
	if err != nil {
		return fmt.Errorf("error fetching object '%s': %v", object, err)
	}

	// Write the object data to a YAML file
	objectFilePath := filepath.Join(objectDir, objectName+".yaml")
	if err := writeObject(objectData, objectFilePath); err != nil {
		return fmt.Errorf("error writing object '%s': %v", objectName, err)
	}
	return nil
}

func getObject(dynamicClient dynamic.Interface, ns string, resource schema.GroupVersionResource, object string) ([]byte, string, error) {
	// Fetch the object
	objectData, err := dynamicClient.Resource(resource).Namespace(ns).Get(context.TODO(), object, metav1.GetOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("error fetching object '%s': %v", object, err)
	}

	// Extract object name and marshal data to JSON
	objectName := objectData.GetName()
	objectJSON, err := json.Marshal(objectData.Object)
	if err != nil {
		return nil, "", fmt.Errorf("error converting object '%s' to JSON: %v", object, err)
	}
	return objectJSON, objectName, nil
}

func writeObject(objectData []byte, objectFile string) error {
	// Create the file
	f, err := os.Create(objectFile)
	if err != nil {
		return fmt.Errorf("error creating file '%s': %v", objectFile, err)
	}
	defer f.Close()

	// Write the data to the file
	if _, err := f.Write(objectData); err != nil {
		return fmt.Errorf("error writing to file '%s': %v", objectFile, err)
	}

	return nil
}

// CompressBackup creates a tarball of the source directory and compresses it using gzip.
func CompressBackup(srcDir string) (string, error) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	tarFilePath := filepath.Join("/tmp/", fmt.Sprintf("kubebackup_%s.tar.gz", timestamp))

	log.Debugf("Starting compression process for directory: %s", srcDir)
	log.Debugf("Generated tarball file path: %s", tarFilePath)

	// Create the tarball
	log.Infof("Creating tarball at: %s", tarFilePath)
	err := createTarball(srcDir, tarFilePath)
	if err != nil {
		log.Errorf("Error during tarball creation for directory %s: %v", srcDir, err)
		return "", fmt.Errorf("error creating tarball: %v", err)
	}

	log.Infof("Successfully created tarball: %s", tarFilePath)
	return tarFilePath, nil
}

func createTarball(srcDir, tarFilePath string) error {
	tarFile, err := os.Create(tarFilePath)
	if err != nil {
		return fmt.Errorf("error creating tar file: %v", err)
	}
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(srcDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking file path: %v", err)
		}

		// Skip the tarball file itself
		if file == tarFilePath {
			return nil
		}

		// Create a tar header
		relPath, err := filepath.Rel(srcDir, file)
		if err != nil {
			return fmt.Errorf("error calculating relative path: %v", err)
		}

		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return fmt.Errorf("error creating tar header: %v", err)
		}
		header.Name = relPath

		// Write the header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("error writing tar header: %v", err)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Write the file content
		fileHandle, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
		defer fileHandle.Close()

		if _, err := io.Copy(tarWriter, fileHandle); err != nil {
			return fmt.Errorf("error writing file content to tar: %v", err)
		}

		return nil
	})
}

func CleanupTmpDir(tmpDir string) error {
	log.Infof("Cleaning up temporary directory: %s", tmpDir)

	// Remove the temporary directory and its contents
	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("error cleaning up temporary directory '%s': %v", tmpDir, err)
	}

	log.Infof("Temporary directory %s cleaned up successfully.", tmpDir)
	return nil
}
