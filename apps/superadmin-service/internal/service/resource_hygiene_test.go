package superadmin

import (
	"testing"
	"time"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAggregateDormantResourceRows(t *testing.T) {
	createdAt := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	laterUpdatedAt := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)

	owners := aggregateDormantResourceRows([]dormantResourceQueryRow{
		{
			UserID:                "user-1",
			OrganizationID:        "org-1",
			VPSCount:              1,
			VPSDiskBytes:          50,
			LastResourceCreatedAt: &createdAt,
			LastResourceUpdatedAt: &updatedAt,
		},
		{
			UserID:                "user-1",
			OrganizationID:        "org-1",
			DatabaseCount:         2,
			DatabaseDiskBytes:     150,
			LastResourceCreatedAt: &createdAt,
			LastResourceUpdatedAt: &laterUpdatedAt,
		},
		{
			UserID:                 "user-1",
			OrganizationID:         "org-2",
			DeploymentCount:        1,
			DeploymentStorageBytes: 20,
			LastResourceCreatedAt:  &updatedAt,
			LastResourceUpdatedAt:  &updatedAt,
		},
	})

	owner := owners["user-1"]
	if owner == nil {
		t.Fatalf("expected aggregate for user-1")
	}
	if owner.VPSCount != 1 || owner.DatabaseCount != 2 || owner.DeploymentCount != 1 {
		t.Fatalf("unexpected resource counts: %+v", owner)
	}
	if owner.TotalReservedBytes != 220 {
		t.Fatalf("expected reserved bytes 220, got %d", owner.TotalReservedBytes)
	}
	if len(owner.Organizations) != 2 {
		t.Fatalf("expected 2 organizations, got %d", len(owner.Organizations))
	}
	if owner.LastResourceUpdatedAt == nil || !owner.LastResourceUpdatedAt.Equal(laterUpdatedAt) {
		t.Fatalf("expected latest updated time %s, got %+v", laterUpdatedAt, owner.LastResourceUpdatedAt)
	}
}

func TestChooseDormantOwnerLastActivity(t *testing.T) {
	profileCreatedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	profileUpdatedAt := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	resourceUpdatedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	auditAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	profile := &authv1.User{
		Id:        "user-1",
		CreatedAt: timestamppb.New(profileCreatedAt),
		UpdatedAt: timestamppb.New(profileUpdatedAt),
	}

	lastActivityAt, source := chooseDormantOwnerLastActivity(profile, &auditAt, &resourceUpdatedAt, nil)
	if source != "audit_log" {
		t.Fatalf("expected audit_log source, got %s", source)
	}
	if lastActivityAt == nil || !lastActivityAt.Equal(auditAt) {
		t.Fatalf("expected audit timestamp %s, got %+v", auditAt, lastActivityAt)
	}

	lastActivityAt, source = chooseDormantOwnerLastActivity(profile, nil, &resourceUpdatedAt, nil)
	if source != "profile_updated" {
		t.Fatalf("expected profile_updated source, got %s", source)
	}
	if lastActivityAt == nil || !lastActivityAt.Equal(profileUpdatedAt) {
		t.Fatalf("expected profile updated timestamp %s, got %+v", profileUpdatedAt, lastActivityAt)
	}

	lastActivityAt, source = chooseDormantOwnerLastActivity(nil, nil, &resourceUpdatedAt, nil)
	if source != "resource_updated" {
		t.Fatalf("expected resource_updated source, got %s", source)
	}
	if lastActivityAt == nil || !lastActivityAt.Equal(resourceUpdatedAt) {
		t.Fatalf("expected resource updated timestamp %s, got %+v", resourceUpdatedAt, lastActivityAt)
	}
}

func TestMatchesDormantResourceSearch(t *testing.T) {
	owner := &superadminv1.DormantResourceOwner{
		User: &superadminv1.UserInfo{
			Id:                "user-1",
			Email:             "owner@example.com",
			Name:              "Example Owner",
			PreferredUsername: "example-owner",
		},
		Organizations: []*superadminv1.DormantResourceOrganization{
			{
				OrganizationId:   "org-1",
				OrganizationName: "Dormant Org",
			},
		},
	}

	if !matchesDormantResourceSearch(owner, "dormant") {
		t.Fatalf("expected organization search to match")
	}
	if !matchesDormantResourceSearch(owner, "example-owner") {
		t.Fatalf("expected username search to match")
	}
	if matchesDormantResourceSearch(owner, "nope") {
		t.Fatalf("did not expect unrelated search term to match")
	}
}
