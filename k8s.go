package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func k8sClientFromKubeconfig(kubeconfigPath string) (*kubernetes.Clientset, error) {
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config from kubeconfig path %s: %w", kubeconfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s client from config: %w", err)
	}

	return clientset, nil
}

func k8sClientInCluster() (*kubernetes.Clientset, error) {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s client from config: %w", err)
	}

	return clientset, nil
}

func k8sClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	if kubeconfigPath != "" {
		return k8sClientFromKubeconfig(kubeconfigPath)
	}

	return k8sClientInCluster()
}

func listPods(ctx context.Context, client *kubernetes.Clientset) (*corev1.PodList, error) {
	coreV1API := client.CoreV1()
	return coreV1API.Pods("").List(ctx, metav1.ListOptions{})
}
