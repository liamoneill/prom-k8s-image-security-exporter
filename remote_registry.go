package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/distribution/distribution/reference"
	"github.com/sirupsen/logrus"
)

type ImageInspection struct {
	Created time.Time `json:"Created"`
}

type RemoteRegistry interface {
	Inspect(ctx context.Context, image reference.Reference) (*ImageInspection, error)
}

type skopeoRemoteRegistry struct {
	logger *logrus.Logger
}

func (r *skopeoRemoteRegistry) Inspect(ctx context.Context, image reference.Reference) (*ImageInspection, error) {
	args := []string{
		"inspect",
		"--override-os",
		"linux",
	}

	ecrToken, err := ecrCreditionalsForImage(ctx, image)
	if err != nil {
		if !errors.Is(err, errUnrecognisedECRImage) {
			return nil, fmt.Errorf("error getting ecr creditionals: %w", err)
		}
	} else {
		creds, err := base64.StdEncoding.DecodeString(ecrToken)
		if err != nil {
			return nil, err
		}
		args = append(args, "--creds", string(creds))
	}

	args = append(args, fmt.Sprintf("docker://%s", image.String()))

	cmd := append([]string{"skopeo"}, args...)
	for i := range cmd {
		if i > 0 && cmd[i-1] == "--creds" {
			cmd[i] = "<redacted>"
		}
	}
	r.logger.WithField("cmd", cmd).Info("running command")

	stdout, err := exec.CommandContext(ctx, "skopeo", args...).Output()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("error running command %v: %w, stderr: [%s]", cmd, err, string(exitErr.Stderr))
	} else if err != nil {
		return nil, fmt.Errorf("error running command %v: %w", cmd, err)
	}

	inspectionResult := ImageInspection{}
	if err := json.Unmarshal(stdout, &inspectionResult); err != nil {
		return nil, fmt.Errorf("error decoding output as json: %w", err)
	}

	return &inspectionResult, nil
}
