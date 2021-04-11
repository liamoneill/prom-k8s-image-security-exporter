package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	severities = []string{
		ecr.FindingSeverityInformational,
		ecr.FindingSeverityLow,
		ecr.FindingSeverityMedium,
		ecr.FindingSeverityHigh,
		ecr.FindingSeverityCritical,
		ecr.FindingSeverityUndefined,
	}
)

type imageScanFindingsMetricsCollector struct {
	k8sClient   *kubernetes.Clientset
	logger      *logrus.Logger
	imageParser ImageParser
}

func (c *imageScanFindingsMetricsCollector) Collect(ctx context.Context) error {
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
		logger := c.logger.WithField("image", image)

		ref, err := c.imageParser.ParseK8SImageID(image)
		if err != nil {
			logger.WithError(err).Warn("skipping image, unexpected error while parsing")
			continue
		}

		if !c.imageParser.MatchesECRFilter(ref) {
			logger.Info("skipping image as it does not match filter")
			continue
		}

		findings, err := GetImageScanFindings(ctx, ref)
		if err != nil {
			logger.WithError(err).Warn("skipping image, unexpected error while getting scan findings")
			continue
		}

		for _, severity := range severities {
			count := 0
			if countPtr, ok := findings.FindingSeverityCounts[severity]; ok {
				count = int(*countPtr)
			}

			imageVulnerabilitiesGauge.With(prometheus.Labels{
				"image":    ref.String(),
				"severity": strings.ToLower(severity),
			}).Set(float64(count))
		}

		logger.Info("successfully retrieved image scan findings")
	}

	return nil
}
