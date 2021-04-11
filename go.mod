module github.com/oneillliam/k8s-image-exporter

go 1.16

require (
	github.com/aws/aws-sdk-go v1.27.0
	github.com/distribution/distribution v2.7.1+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/gorilla/mux v1.8.0
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
)
