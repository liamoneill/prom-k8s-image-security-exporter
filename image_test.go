package main

import (
	"testing"

	"github.com/distribution/distribution/reference"
	"github.com/stretchr/testify/assert"
)

func TestParseK8SImageID(t *testing.T) {
	imageID := "docker-pullable://602401143452.dkr.ecr.us-east-2.amazonaws.com/amazon-k8s-cni@sha256:bf321746d8a281e6a1437cb2b008953be2a729773938fa759c02ee2e9ba140b7"

	imageParser, _ := newImageParser("^602401143452.dkr.ecr.us-east-2.amazonaws.com/")
	ref, err := imageParser.ParseK8SImageID(imageID)

	assert.Nil(t, err)
	canonical, ok := ref.(reference.Canonical)
	if !ok {
		assert.Fail(t, "expected type reference.Canonical")
	}

	assert.Equal(t, "602401143452.dkr.ecr.us-east-2.amazonaws.com/amazon-k8s-cni", canonical.Name())
	assert.Equal(t, "602401143452.dkr.ecr.us-east-2.amazonaws.com", reference.Domain(canonical))
	assert.Equal(t, "amazon-k8s-cni", reference.Path(canonical))

	assert.True(t, imageParser.MatchesECRFilter(ref))
}

func TestParseK8SImageID2(t *testing.T) {
	imageID := "docker-pullable://nginx@sha256:bda886ac14a4dee943636e1a48b3280616bad42698a019ef21f48092a52c5b13"

	imageParser, _ := newImageParser("")
	ref, err := imageParser.ParseK8SImageID(imageID)

	assert.Nil(t, err)
	canonical, ok := ref.(reference.Canonical)
	if !ok {
		assert.Fail(t, "expected type reference.Canonical")
	}

	assert.Equal(t, "nginx", canonical.Name())
	assert.Equal(t, "", reference.Domain(canonical))
	assert.Equal(t, "nginx", reference.Path(canonical))

	assert.False(t, imageParser.MatchesECRFilter(ref))
}
