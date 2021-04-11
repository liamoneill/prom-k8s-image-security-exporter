package main

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type imageMetricsCollector struct {
	logger         *logrus.Logger
	k8sClient      *kubernetes.Clientset
	imageParser    ImageParser
	remoteRegistry RemoteRegistry
}

func (c *imageMetricsCollector) Collect(ctx context.Context) error {
	coreV1API := c.k8sClient.CoreV1()
	pods, err := coreV1API.Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing pods: %w", err)
	}

	imageSet := make(map[string]bool)
	for _, pod := range pods.Items {
		for _, container := range pod.Status.ContainerStatuses {
			imageSet[container.ImageID] = true
		}
	}

	for image := range imageSet {
		ref, err := c.imageParser.ParseK8SImageID(image)
		if err != nil {
			c.logger.WithError(err).Warnf("skipping image [%s], unexpected error while parsing", image)
			continue
		}

		inspection, err := c.remoteRegistry.Inspect(ctx, ref)
		if err != nil {
			c.logger.WithError(err).Warnf("skipping image [%s], unexpected error while inspecting image in remote registry", ref.String())
			continue
		}

		imageAge := float64(time.Now().Sub(inspection.Created)) / float64(24*time.Hour)
		imageAgeDaysGauge.With(prometheus.Labels{
			"image": ref.String(),
		}).Set(imageAge)
	}

	return nil
}
