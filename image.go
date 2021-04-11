package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/distribution/distribution/reference"
)

type Image = reference.Reference

type ImageParser interface {
	ParseK8SImageID(imageID string) (Image, error)
	MatchesECRFilter(image Image) bool
}

type imageParser struct {
	ecrFilter *regexp.Regexp
}

func newImageParser(ecrFilter string) (ImageParser, error) {
	if ecrFilter == "" {
		ecrFilter = "^$"
	}

	ecrFilterRegexp, err := regexp.Compile(ecrFilter)
	if err != nil {
		return nil, fmt.Errorf("cannot compile ecr regex filter [%s]: %w", ecrFilter, err)
	}

	return &imageParser{
		ecrFilter: ecrFilterRegexp,
	}, nil
}

func (*imageParser) ParseK8SImageID(imageID string) (Image, error) {
	if matched, _ := regexp.MatchString(`^docker-pullable://.*@sha256:.*$`, imageID); !matched {
		return nil, fmt.Errorf("unrecognised imageID [%s]", imageID)
	}

	image := strings.Replace(imageID, "docker-pullable://", "", 1)
	ref, err := reference.Parse(image)
	if err != nil {
		return nil, fmt.Errorf("could not parse image [%s]: %w", image, err)
	}

	return ref, nil
}

func (p *imageParser) MatchesECRFilter(image Image) bool {
	named, ok := image.(reference.Named)
	if !ok {
		return false
	}

	return p.ecrFilter.MatchString(named.Name())
}
