package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
)

var (
	// The `gitRevision` variable is optionally set at compilation time.
	// See the Dockerfile for more info.
	gitRevision string

	listenAddress        string
	listenPort           int
	ecrScanResultPattern string
	kubeconfigPath       string
)

func init() {
	flag.StringVar(&listenAddress, "address", "0.0.0.0", "Address to listen on")
	flag.IntVar(&listenPort, "port", 5000, "Port to listen on")
	flag.StringVar(&ecrScanResultPattern, "ecr-scan-results-filter", "^$", "Regular expression pattern to filter out ECR repos")
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig for running the service outside the cluster in development mode")
}

type cronLoggerAdapter struct {
	logger *logrus.Logger
}

func (l *cronLoggerAdapter) Info(msg string, keysAndValues ...interface{}) {
	// discard info messages to reduce verbosity
}

func (l *cronLoggerAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	logger := l.logger

	logger.WithError(err).
		WithField("fields", keysAndValues).
		Error(msg)

	cronErrorsCounter.Inc()
}

func newCron(logger *logrus.Logger, imageMetricsCollector *imageMetricsCollector, imageScanFindingsMetricsCollector *imageScanFindingsMetricsCollector) *cron.Cron {
	cronLogger := &cronLoggerAdapter{logger: logger}
	c := cron.New(
		cron.WithLocation(time.UTC),
		cron.WithChain(
			cron.SkipIfStillRunning(cronLogger),
		))

	c.AddFunc("23 * * * *", func() {
		if err := imageMetricsCollector.Collect(context.TODO()); err != nil {
			logger.WithError(err).Info("failed to collect image metrics")
		}
	})

	c.AddFunc("53 * * * *", func() {
		if err := imageScanFindingsMetricsCollector.Collect(context.TODO()); err != nil {
			logger.WithError(err).Info("failed to collect image scan findings metrics")
		}
	})

	return c
}

func newHealthcheckHandler(logger *logrus.Logger, k8sClient *kubernetes.Clientset) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := listPods(context.TODO(), k8sClient); err != nil {
			logger.WithError(err).Error("failed healthcheck")

			w.WriteHeader(500)
			fmt.Fprintf(w, "error listing pods from k8s api: %s", err)
			return
		}

		fmt.Fprint(w, "ok")
	})
}

func newRefreshCollectorHandler(logger *logrus.Logger, collector Collector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := collector.Collect(context.TODO()); err != nil {
			logger.WithError(err).Error("failed to refresh metrics")

			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to refresh metrics: %s", err)
			return
		}

		fmt.Fprint(w, "ok")
	})
}

func httpHandler(logger *logrus.Logger, k8sClient *kubernetes.Clientset, imageMetricsCollector *imageMetricsCollector, imageScanFindingsMetricsCollector *imageScanFindingsMetricsCollector) http.Handler {
	router := mux.NewRouter()

	router.Methods("GET").Path("/health").Handler(newHealthcheckHandler(logger, k8sClient))
	router.Methods("GET").Path("/metrics").Handler(promhttp.Handler())

	router.Methods("POST").Path("/refresh-metrics/images").Handler(newRefreshCollectorHandler(logger, imageMetricsCollector))
	router.Methods("POST").Path("/refresh-metrics/scan-findings").Handler(newRefreshCollectorHandler(logger, imageScanFindingsMetricsCollector))

	return router
}

func main() {
	flag.Parse()

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.WithFields(logrus.Fields{
		"gitRevison":           gitRevision,
		"listenAddress":        listenAddress,
		"listenPort":           listenPort,
		"ecrScanResultPattern": ecrScanResultPattern,
		"kubeconfigPath":       kubeconfigPath,
	}).Info("loaded config")

	k8sClient, err := k8sClient(kubeconfigPath)
	if err != nil {
		logger.WithError(err).Fatal("could not create k8s client")
	}

	skopeoRemoteRegistry := &skopeoRemoteRegistry{
		logger: logger,
	}
	imageParser, _ := newImageParser(ecrScanResultPattern)
	imageMetricsCollector := &imageMetricsCollector{
		logger:         logger,
		k8sClient:      k8sClient,
		imageParser:    imageParser,
		remoteRegistry: skopeoRemoteRegistry,
	}

	imageScanFindingsMetricsCollector := &imageScanFindingsMetricsCollector{
		logger:      logger,
		k8sClient:   k8sClient,
		imageParser: imageParser,
	}

	logger.Info("starting background metric collection")
	c := newCron(logger, imageMetricsCollector, imageScanFindingsMetricsCollector)
	c.Start()

	address := fmt.Sprintf("%s:%d", listenAddress, listenPort)
	logger.WithField("listen_address", address).Info("starting server")

	handler := httpHandler(logger, k8sClient, imageMetricsCollector, imageScanFindingsMetricsCollector)
	err = http.ListenAndServe(address, handler)
	logger.WithError(err).Fatal("server unexpectedly exited")
}
