package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MaxDelegatedDNSTTL int64 = 3600

var ErrDNSDelegationForbidden = errors.New("DNS delegation is not authorized for this organization or domain")
var ErrInvalidDelegatedDNSRecord = errors.New("invalid delegated DNS record")

func normalizeDelegatedDomain(domain string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
}

func delegatedResourceDomain(domain string) string {
	parts := strings.SplitN(domain, ".", 3)
	if len(parts) == 3 && strings.HasPrefix(parts[0], "_") && strings.HasPrefix(parts[1], "_") {
		return parts[2]
	}
	return domain
}

func validateDelegatedRecord(domain, recordType, recordsJSON string, ttl int64) error {
	if ttl < 1 || ttl > MaxDelegatedDNSTTL || len(domain) > 253 || (recordType != "A" && recordType != "SRV") {
		return ErrInvalidDelegatedDNSRecord
	}
	base := delegatedResourceDomain(domain)
	if recordType == "SRV" {
		if base == domain || !(strings.Contains(domain, "._tcp.") || strings.Contains(domain, "._udp.")) {
			return ErrInvalidDelegatedDNSRecord
		}
	} else if base != domain {
		return ErrInvalidDelegatedDNSRecord
	}
	for i, label := range strings.Split(domain, ".") {
		if recordType == "SRV" && i < 2 {
			label = strings.TrimPrefix(label, "_")
		}
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return ErrInvalidDelegatedDNSRecord
		}
		for _, ch := range label {
			if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
				return ErrInvalidDelegatedDNSRecord
			}
		}
	}
	var records []string
	if json.Unmarshal([]byte(recordsJSON), &records) != nil || len(records) == 0 || len(records) > 100 {
		return ErrInvalidDelegatedDNSRecord
	}
	for _, value := range records {
		if recordType == "A" {
			if ip := net.ParseIP(value); ip == nil || ip.To4() == nil {
				return ErrInvalidDelegatedDNSRecord
			}
			continue
		}
		fields := strings.Fields(value)
		if len(fields) != 4 || normalizeDelegatedDomain(fields[3]) != base {
			return ErrInvalidDelegatedDNSRecord
		}
		for _, value := range fields[:3] {
			if _, err := strconv.ParseUint(value, 10, 16); err != nil {
				return ErrInvalidDelegatedDNSRecord
			}
		}
	}
	return nil
}

func activeDelegationOrganization(tx *gorm.DB, orgID string, lock bool) error {
	if orgID == "" {
		return ErrDNSDelegationForbidden
	}
	var org Organization
	query := tx
	if lock {
		query = query.Clauses(clause.Locking{Strength: "SHARE"})
	}
	if err := query.First(&org, "id = ?", orgID).Error; err != nil {
		return err
	}
	status := strings.ToLower(org.Status)
	if status == "active" || (status == "suspended" && org.SuspensionExpires != nil && !org.SuspensionExpires.After(time.Now())) {
		return nil
	}
	return ErrDNSDelegationForbidden
}

// A remote installation may claim unused resource hostnames. It may never
// replace a local resource or another organization's existing delegation.
// Custom domains require verification in the organization that owns the key.
func authorizeDelegationDomain(tx *gorm.DB, domain, orgID string) error {
	base := delegatedResourceDomain(domain)
	const suffix = ".my.obiente.cloud"
	if strings.HasSuffix(base, suffix) {
		id := strings.TrimSuffix(base, suffix)
		if strings.Contains(id, ".") {
			return ErrDNSDelegationForbidden
		}
		var model any
		switch {
		case strings.HasPrefix(id, "deploy-"):
			model = &Deployment{}
		case strings.HasPrefix(id, "gs-"):
			model = &GameServer{}
		case strings.HasPrefix(id, "db-"):
			model = &DatabaseInstance{}
		default:
			return ErrDNSDelegationForbidden
		}
		var count int64
		query := tx.Model(model).Where("LOWER(id) = ? OR LOWER(REPLACE(id, '-', '')) LIKE ?", id, strings.ReplaceAll(id, "-", "")+"%")
		if _, deployment := model.(*Deployment); deployment {
			query = query.Or("LOWER(RTRIM(domain, '.')) = ?", base)
		}
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return ErrDNSDelegationForbidden
		}
		return nil
	}
	var deployments []Deployment
	if err := tx.Where("organization_id = ? AND deleted_at IS NULL", orgID).Find(&deployments).Error; err != nil {
		return err
	}
	for _, deployment := range deployments {
		var domains []string
		if json.Unmarshal([]byte(deployment.CustomDomains), &domains) != nil {
			continue
		}
		for _, entry := range domains {
			parts := strings.Split(entry, ":")
			verified := (len(parts) == 2 && parts[1] == "verified") || (len(parts) == 4 && parts[1] == "token" && parts[3] == "verified")
			if verified && normalizeDelegatedDomain(parts[0]) == base {
				return nil
			}
		}
	}
	return ErrDNSDelegationForbidden
}

func authorizeDelegationWrite(tx *gorm.DB, domain, apiKeyID, orgID string) error {
	// Locally mirrored records are not accepted by the HTTP key-authenticated API.
	if apiKeyID == "" && orgID == "" {
		return nil
	}
	if apiKeyID == "" || orgID == "" {
		return ErrDNSDelegationForbidden
	}
	var key DNSDelegationAPIKey
	if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ? AND organization_id = ? AND is_active = ? AND revoked_at IS NULL", apiKeyID, orgID, true).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDNSDelegationForbidden
		}
		return fmt.Errorf("validate delegation key: %w", err)
	}
	if err := activeDelegationOrganization(tx, orgID, true); err != nil {
		return err
	}
	return authorizeDelegationDomain(tx, domain, orgID)
}
