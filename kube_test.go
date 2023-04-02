package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestKubeFunctions(t *testing.T) {
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

	// Test the processNamespace function
	config := &rest.Config{}
	namespacedObjects, err := getNamespacedObjects(clientset)
	if err != nil {
		t.Fatalf("Error getting namespaced objects: %v", err)
	}
	processNamespace(clientset, config, namespacedObjects, "test-namespace", tmpDir)

	// Test the processClusterScopedObjects function
	processClusterScopedObjects(clientset, config, tmpDir)

	// Check if the output directories and files were created
	namespaceDir := filepath.Join(tmpDir, "namespace", "test-namespace")
	_, err = os.Stat(namespaceDir)
	if os.IsNotExist(err) {
		t.Fatalf("Namespace directory was not created: %v", err)
	}

	clusterDir := filepath.Join(tmpDir, "clusterobjects")
	_, err = os.Stat(clusterDir)
	if os.IsNotExist(err) {
		t.Fatalf("Cluster-scoped objects directory was not created: %v", err)
	}
}
