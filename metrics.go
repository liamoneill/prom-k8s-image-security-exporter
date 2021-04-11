package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	imageAgeDaysGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solve_k8s_image_exporter_image_age_days",
			Help: "Days since the image was built as determined by the image's `Created` field from the image manifest",
		},
		[]string{"image"},
	)
	imageVulnerabilitiesGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solve_k8s_image_exporter_image_vulnerabilities",
			Help: "Days since the image was built as determined by the image's `Created` field from the image manifest",
		},
		[]string{"image", "severity"},
	)
	cronErrorsCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "solve_k8s_image_exporter_cron_errors",
			Help: "Total count of errors from this service's cron library",
		},
	)
)
