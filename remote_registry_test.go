package main

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRemoteRegistry(t *testing.T) {
	if _, ok := os.LookupEnv("RUN_ACCEPTANCE_TESTS"); !ok {
		t.Skip()
	}

	imageID := "docker-pullable://k8s.gcr.io/coredns@sha256:73ca82b4ce829766d4f1f10947c3a338888f876fbed0540dc849c89ff256e90c"

	remoteRegistry := &skopeoRemoteRegistry{
		logger: &logrus.Logger{},
	}
	imageParser, _ := newImageParser("")
	ref, _ := imageParser.ParseK8SImageID(imageID)

	inspection, err := remoteRegistry.Inspect(context.TODO(), ref)
	assert.NoError(t, err)
	assert.NotNil(t, inspection)
}
