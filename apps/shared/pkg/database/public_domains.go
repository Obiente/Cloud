package database

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultPublicDomainSuffix = "my.obiente.cloud"
	shortPublicIDSuffixLength = 16
)

// DefaultMyObienteCloudLabel returns the stable public-facing label used for
// default *.my.obiente.cloud hostnames. UUID-backed IDs are shortened to keep
// hostnames readable without changing the underlying primary key.
func DefaultMyObienteCloudLabel(resourceID string) string {
	resourceID = strings.TrimSpace(strings.ToLower(resourceID))
	prefix, rawID, ok := strings.Cut(resourceID, "-")
	if !ok || prefix == "" || rawID == "" {
		return resourceID
	}

	parsed, err := uuid.Parse(rawID)
	if err != nil {
		return resourceID
	}

	compact := strings.ReplaceAll(parsed.String(), "-", "")
	if len(compact) < shortPublicIDSuffixLength {
		return resourceID
	}

	return fmt.Sprintf("%s-%s", prefix, compact[:shortPublicIDSuffixLength])
}

func DefaultMyObienteCloudDomain(resourceID string) string {
	label := DefaultMyObienteCloudLabel(resourceID)
	if label == "" {
		return ""
	}
	return fmt.Sprintf("%s.%s", label, defaultPublicDomainSuffix)
}

func NormalizeDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}

func ExtractDefaultPublicLabel(domain string) string {
	domain = NormalizeDomain(domain)
	suffix := "." + defaultPublicDomainSuffix
	if !strings.HasSuffix(domain, suffix) {
		return ""
	}
	return strings.TrimSuffix(domain, suffix)
}

func ResolveDeploymentIDByDomain(domain string) (string, error) {
	domain = NormalizeDomain(domain)
	if domain == "" {
		return "", gorm.ErrRecordNotFound
	}

	var deployment Deployment
	if err := DB.Select("id").Where("domain = ? AND deleted_at IS NULL", domain).First(&deployment).Error; err == nil {
		return deployment.ID, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	label := ExtractDefaultPublicLabel(domain)
	if label == "" {
		return "", gorm.ErrRecordNotFound
	}

	return resolveIDByPublicLabel("deployments", "id", "deleted_at", label)
}

func ResolveDatabaseIDByLabel(label string) (string, error) {
	return resolveIDByPublicLabel("database_instances", "id", "deleted_at", label)
}

func ResolveGameServerIDByLabel(label string) (string, error) {
	return resolveIDByPublicLabel("game_servers", "id", "deleted_at", label)
}

func resolveIDByPublicLabel(tableName, idColumn, deletedAtColumn, label string) (string, error) {
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return "", gorm.ErrRecordNotFound
	}

	ids, err := selectIDsByLabel(tableName, idColumn, deletedAtColumn, label)
	if err != nil {
		return "", err
	}
	switch len(ids) {
	case 0:
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("multiple resources matched label %q", label)
	}

	prefix, _, ok := strings.Cut(label, "-")
	if !ok || prefix == "" {
		return "", gorm.ErrRecordNotFound
	}

	candidateIDs, err := selectIDsByPrefix(tableName, idColumn, deletedAtColumn, prefix)
	if err != nil {
		return "", err
	}

	matches := make([]string, 0, 1)
	for _, candidateID := range candidateIDs {
		if DefaultMyObienteCloudLabel(candidateID) == label {
			matches = append(matches, candidateID)
		}
	}

	switch len(matches) {
	case 0:
		return "", gorm.ErrRecordNotFound
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple resources matched public label %q", label)
	}
}

func selectIDsByLabel(tableName, idColumn, deletedAtColumn, label string) ([]string, error) {
	query := DB.Table(tableName).Select(idColumn).Where(fmt.Sprintf("%s = ?", idColumn), label)
	if deletedAtColumn != "" {
		query = query.Where(fmt.Sprintf("%s IS NULL", deletedAtColumn))
	}

	var ids []string
	if err := query.Pluck(idColumn, &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func selectIDsByPrefix(tableName, idColumn, deletedAtColumn, prefix string) ([]string, error) {
	query := DB.Table(tableName).Select(idColumn).Where(fmt.Sprintf("%s LIKE ?", idColumn), prefix+"-%")
	if deletedAtColumn != "" {
		query = query.Where(fmt.Sprintf("%s IS NULL", deletedAtColumn))
	}

	var ids []string
	if err := query.Pluck(idColumn, &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
