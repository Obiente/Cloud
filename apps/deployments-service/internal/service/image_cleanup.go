package deployments

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/platform"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

var revisionImageTagPattern = regexp.MustCompile(`-[a-fA-F0-9]{40}(?:[a-fA-F0-9]{24})?$`)

const registryManifestAccept = "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json"

// retainedRevisionImages keeps the currently deployed image, one prior image
// that passed runtime verification, active build images, and stable mutable
// tags. Older build records remain available without retaining every immutable
// revision tag.
func retainedRevisionImages(builds []*database.BuildHistory, currentImage string) (map[string]struct{}, []string) {
	kept := make(map[string]struct{})
	if currentImage = strings.TrimSpace(currentImage); currentImage != "" {
		kept[currentImage] = struct{}{}
	}
	rollbackRetained := false
	for _, build := range builds {
		if build.ImageName == nil {
			continue
		}
		image := strings.TrimSpace(*build.ImageName)
		if image == "" {
			continue
		}

		// Mutable branch tags do not accumulate and may share a manifest with
		// an immutable revision tag, so never delete their backing manifest.
		if !isRevisionTaggedImage(image) {
			kept[image] = struct{}{}
			continue
		}
		// A preceding cleanup can overlap the next build after the lease is
		// released. Protect any image already produced by that active build.
		if build.Status == int32(deploymentsv1.BuildStatus_BUILD_PENDING) ||
			build.Status == int32(deploymentsv1.BuildStatus_BUILD_BUILDING) {
			kept[image] = struct{}{}
			continue
		}
		// Runtime startup failures deliberately retain BUILD_SUCCESS with an
		// error for build-history accuracy, but are not rollback candidates.
		if build.Status != int32(deploymentsv1.BuildStatus_BUILD_SUCCESS) || build.Error != nil || rollbackRetained {
			continue
		}
		if _, current := kept[image]; current {
			continue
		}
		kept[image] = struct{}{}
		rollbackRetained = true
	}

	obsoleteSet := make(map[string]struct{})
	for _, build := range builds {
		if build.ImageName == nil {
			continue
		}
		image := strings.TrimSpace(*build.ImageName)
		if !isRevisionTaggedImage(image) {
			continue
		}
		if _, retain := kept[image]; !retain {
			obsoleteSet[image] = struct{}{}
		}
	}

	obsolete := make([]string, 0, len(obsoleteSet))
	for image := range obsoleteSet {
		obsolete = append(obsolete, image)
	}
	return kept, obsolete
}

func isRevisionTaggedImage(image string) bool {
	lastSlash := strings.LastIndexByte(image, '/')
	lastColon := strings.LastIndexByte(image, ':')
	return lastColon > lastSlash && revisionImageTagPattern.MatchString(image[lastColon+1:])
}

func cleanupCallerOwnsLiveImage(callerImage string, liveImage *string) bool {
	return liveImage != nil && strings.TrimSpace(callerImage) != "" && strings.TrimSpace(callerImage) == strings.TrimSpace(*liveImage)
}

func (s *Service) cleanupObsoleteRevisionImages(ctx context.Context, deploymentID, organizationID, callerImage string) {
	// Read through the database instead of the deployment repository cache. A
	// cleanup can be delayed while newer revisions finish, and only the actual
	// live image may determine which preceding revision is rollback-safe.
	var deployment database.Deployment
	if err := database.DB.WithContext(ctx).
		Select("image").
		Where("id = ? AND deleted_at IS NULL", deploymentID).
		First(&deployment).Error; err != nil {
		logger.Warn("[ImageCleanup] Failed to read live image for deployment %s: %v", deploymentID, err)
		return
	}
	if !cleanupCallerOwnsLiveImage(callerImage, deployment.Image) {
		logger.Info("[ImageCleanup] Skipping stale cleanup for deployment %s because a newer image is live", deploymentID)
		return
	}
	currentImage := strings.TrimSpace(*deployment.Image)

	builds, _, err := s.buildHistoryRepo.ListBuilds(ctx, deploymentID, organizationID, 0, 0)
	if err != nil {
		logger.Warn("[ImageCleanup] Failed to list build images for deployment %s: %v", deploymentID, err)
		return
	}

	kept, obsolete := retainedRevisionImages(builds, currentImage)
	if len(obsolete) == 0 {
		return
	}

	registryURL := platform.RegistryURL()
	client := &http.Client{Timeout: 15 * time.Second}
	protectedDigests := make(map[string]map[string]struct{})
	unsafeRepositories := make(map[string]struct{})
	for image := range kept {
		repository, tag, ok := splitRegistryImage(registryURL, image)
		if !ok {
			continue
		}
		digest, err := registryManifestDigest(ctx, client, registryURL, repository, tag)
		if err != nil {
			unsafeRepositories[repository] = struct{}{}
			logger.Warn("[ImageCleanup] Keeping registry repository %s unchanged because protected image %s could not be resolved: %v", repository, image, err)
			continue
		}
		if protectedDigests[repository] == nil {
			protectedDigests[repository] = make(map[string]struct{})
		}
		protectedDigests[repository][digest] = struct{}{}
	}

	for _, image := range obsolete {
		if repository, tag, ok := splitRegistryImage(registryURL, image); ok {
			if _, unsafe := unsafeRepositories[repository]; !unsafe {
				digest, err := registryManifestDigest(ctx, client, registryURL, repository, tag)
				if err != nil {
					logger.Warn("[ImageCleanup] Failed to resolve obsolete registry image %s: %v", image, err)
				} else if _, protected := protectedDigests[repository][digest]; protected {
					logger.Info("[ImageCleanup] Retaining registry manifest %s because a rollback tag still references it", digest)
				} else if err := deleteRegistryManifest(ctx, client, registryURL, repository, digest); err != nil {
					logger.Warn("[ImageCleanup] Failed to delete obsolete registry image %s: %v", image, err)
				} else {
					logger.Info("[ImageCleanup] Deleted obsolete registry image %s", image)
				}
			}
		}

		for _, localImage := range localImageAliases(registryURL, image) {
			if err := exec.CommandContext(ctx, "docker", "image", "rm", localImage).Run(); err != nil {
				logger.Debug("[ImageCleanup] Could not remove local image tag %s: %v", localImage, err)
			}
		}
	}
}

func splitRegistryImage(registryURL, image string) (repository, tag string, ok bool) {
	parsed, err := url.Parse(registryURL)
	if err != nil || parsed.Host == "" {
		return "", "", false
	}
	prefix := parsed.Host + "/"
	if !strings.HasPrefix(image, prefix) {
		return "", "", false
	}
	reference := strings.TrimPrefix(image, prefix)
	lastSlash := strings.LastIndexByte(reference, '/')
	lastColon := strings.LastIndexByte(reference, ':')
	if lastColon <= lastSlash || lastColon == len(reference)-1 {
		return "", "", false
	}
	return reference[:lastColon], reference[lastColon+1:], true
}

func registryManifestDigest(ctx context.Context, client *http.Client, registryURL, repository, tag string) (string, error) {
	req, err := registryRequest(ctx, http.MethodHead, registryURL, repository, tag)
	if err != nil {
		return "", err
	}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("registry returned %s", response.Status)
	}
	digest := strings.TrimSpace(response.Header.Get("Docker-Content-Digest"))
	if digest == "" {
		return "", fmt.Errorf("registry response omitted Docker-Content-Digest")
	}
	return digest, nil
}

func deleteRegistryManifest(ctx context.Context, client *http.Client, registryURL, repository, digest string) error {
	req, err := registryRequest(ctx, http.MethodDelete, registryURL, repository, digest)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("registry returned %s", response.Status)
	}
	return nil
}

func registryRequest(ctx context.Context, method, registryURL, repository, reference string) (*http.Request, error) {
	base := strings.TrimSuffix(registryURL, "/")
	segments := strings.Split(repository, "/")
	for index := range segments {
		segments[index] = url.PathEscape(segments[index])
	}
	endpoint := fmt.Sprintf("%s/v2/%s/manifests/%s", base, strings.Join(segments, "/"), url.PathEscape(reference))
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", registryManifestAccept)
	username := strings.TrimSpace(os.Getenv("REGISTRY_USERNAME"))
	if username == "" {
		username = "obiente"
	}
	if password := os.Getenv("REGISTRY_PASSWORD"); password != "" {
		req.SetBasicAuth(username, password)
	}
	return req, nil
}

func localImageAliases(registryURL, image string) []string {
	aliases := []string{image}
	if parsed, err := url.Parse(registryURL); err == nil && parsed.Host != "" {
		if local := strings.TrimPrefix(image, parsed.Host+"/"); local != image {
			aliases = append(aliases, local)
		}
	}
	return aliases
}
