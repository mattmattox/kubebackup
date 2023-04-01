package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

func loadKubeConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func getNamespacedObjects(clientset *kubernetes.Clientset) ([]schema.GroupVersionResource, error) {
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
	return objects, nil
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

			// Increment the counters for specific object types
			switch object.Resource {
			case "pods":
				objectCount.WithLabelValues("pods").Inc()
			case "deployments":
				objectCount.WithLabelValues("deployments").Inc()
			case "statefulsets":
				objectCount.WithLabelValues("statefulsets").Inc()
			case "secrets":
				objectCount.WithLabelValues("secrets").Inc()
			case "configmaps":
				objectCount.WithLabelValues("configmaps").Inc()
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
