package main

import (
	"context"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetNamespacedObjects(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	objects := getNamespacedObjects(clientset)

	for _, object := range objects {
		if !object.Namespaced {
			t.Errorf("Expected namespaced object, got non-namespaced object: %v", object)
		}
	}
}

func TestGetNamespaces(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-namespace",
		},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatal(err)
	}

	namespaces, err := getNamespaces(clientset)
	if err != nil {
		t.Fatal(err)
	}

	expectedNamespace := "test-namespace"
	if len(namespaces) != 1 || namespaces[0] != expectedNamespace {
		t.Errorf("Expected [%v], got %v", expectedNamespace, namespaces)
	}
}

func TestLoadKubeConfig(t *testing.T) {
	_, err := loadKubeConfig("")
	if err != nil {
		t.Errorf("Error loading kubeconfig: %v", err)
	}
}
