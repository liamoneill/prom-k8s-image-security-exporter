package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/distribution/distribution/reference"
)

var (
	ecrRepoRegex = regexp.MustCompile(`^(?P<registryID>[^.]+)\.dkr\.ecr\.(?P<region>[^.]+)\.amazonaws\.com/`)
	maxResults   = int64(1000)

	errUnrecognisedECRImage = errors.New("unrecognised ECR image")

	ecrServicesMutex         = &sync.Mutex{}
	ecrServices              = make(map[string]*ecr.ECR)
	ecrCreditionalCacheMutex = &sync.Mutex{}
	ecrCreditionalCache      = make(map[string]*ecr.AuthorizationData)
)

func ecrService(region string) (*ecr.ECR, error) {
	ecrServicesMutex.Lock()
	defer ecrServicesMutex.Unlock()

	if svc, ok := ecrServices[region]; ok {
		return svc, nil
	}

	sess, err := session.NewSession(&aws.Config{
		Region: &region,
	})
	if err != nil {
		return nil, err
	}

	svc := ecr.New(sess)
	ecrServices[region] = svc

	return svc, nil
}

func ecrCreditionalsForImage(ctx context.Context, image reference.Reference) (string, error) {
	match := ecrRepoRegex.FindStringSubmatch(image.String())
	if len(match) == 0 {
		return "", errUnrecognisedECRImage
	}
	region := match[2]

	svc, err := ecrService(region)
	if err != nil {
		return "", err
	}

	ecrCreditionalCacheMutex.Lock()
	defer ecrCreditionalCacheMutex.Unlock()

	authData, ok := ecrCreditionalCache[region]
	if ok && authData.ExpiresAt.Sub(time.Now()) > 5*time.Minute {
		return *authData.AuthorizationToken, nil
	}

	output, err := svc.GetAuthorizationTokenWithContext(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	}

	authData = output.AuthorizationData[0]
	ecrCreditionalCache[region] = authData
	return *authData.AuthorizationToken, nil
}

func GetImageScanFindings(ctx context.Context, image reference.Reference) (*ecr.ImageScanFindings, error) {
	match := ecrRepoRegex.FindStringSubmatch(image.String())
	if len(match) == 0 {
		return nil, errUnrecognisedECRImage
	}
	registryID := match[1]
	region := match[2]

	var repository string
	if named, ok := image.(reference.Named); ok {
		repository = reference.Path(named)
	} else {
		return nil, fmt.Errorf("unrecognised image [%s], has repository name", image.String())
	}

	var imageID *ecr.ImageIdentifier
	if tagged, ok := image.(reference.Tagged); ok {
		tag := tagged.Tag()
		imageID = &ecr.ImageIdentifier{
			ImageTag: &tag,
		}
	} else if digested, ok := image.(reference.Digested); ok {
		digest := string(digested.Digest())
		imageID = &ecr.ImageIdentifier{
			ImageDigest: &digest,
		}
	} else {
		return nil, fmt.Errorf("unrecognised image [%s], has neither tag nor digest", image.String())
	}

	ecrService, err := ecrService(region)
	if err != nil {
		return nil, fmt.Errorf("error creating ECR service: %w", err)
	}

	output, err := ecrService.DescribeImageScanFindingsWithContext(ctx, &ecr.DescribeImageScanFindingsInput{
		RegistryId:     &registryID,
		RepositoryName: &repository,
		ImageId:        imageID,
		MaxResults:     &maxResults,
	})
	if err != nil {
		return nil, fmt.Errorf("error describing image scan findings for image [%s]: %w", image.String(), err)
	}

	status := *output.ImageScanStatus.Status
	if status != ecr.ScanStatusComplete {
		return nil, fmt.Errorf("image scan did not complete for image [%s], the current status is [%s]", image.String(), status)
	}

	return output.ImageScanFindings, nil
}
