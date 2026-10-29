package organizations

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	organizationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/organizations/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func TestOrganizationCreationRollsBackPartialSetup(t *testing.T) {
	for _, personal := range []bool{false, true} {
		for _, failure := range []string{"organization_members", "org_quotas", "missing_default_plan"} {
			name := failure
			if personal {
				name = "personal_" + name
			}
			t.Run(name, func(t *testing.T) {
				db := newOrganizationServiceTestDB(t)
				if failure != "missing_default_plan" {
					if err := db.Create(&database.OrganizationPlan{ID: "plan-starter", Name: "Starter"}).Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := db.Callback().Create().Before("gorm:create").Register("test:setup_failure", func(tx *gorm.DB) {
					if tx.Statement.Table == failure {
						tx.AddError(errors.New("synthetic setup failure"))
					}
				}); err != nil {
					t.Fatal(err)
				}
				ctx := auth.WithUser(context.Background(), &authv1.User{Id: "new-user", Email: "new@example.test"})
				create := func() error {
					if personal {
						_, err := (&Service{}).ListOrganizations(ctx, connect.NewRequest(&organizationsv1.ListOrganizationsRequest{OnlyMine: proto.Bool(true)}))
						return err
					}
					_, err := (&Service{}).CreateOrganization(ctx, connect.NewRequest(&organizationsv1.CreateOrganizationRequest{Name: "New Organization"}))
					return err
				}
				if err := create(); connect.CodeOf(err) != connect.CodeInternal {
					t.Fatalf("failure=%v", err)
				}
				for _, model := range []any{&database.Organization{}, &database.OrganizationMember{}, &database.OrgQuota{}} {
					var count int64
					if err := db.Model(model).Count(&count).Error; err != nil {
						t.Fatal(err)
					}
					if count != 0 {
						t.Fatalf("partial setup persisted %T: %d", model, count)
					}
				}
				db.Callback().Create().Remove("test:setup_failure")
				if failure == "missing_default_plan" {
					if err := db.Create(&database.OrganizationPlan{ID: "plan-starter", Name: "Starter"}).Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := create(); err != nil {
					t.Fatal(err)
				}
				if personal {
					if err := create(); err != nil {
						t.Fatal(err)
					}
				}
				var org database.Organization
				if err := db.First(&org).Error; err != nil {
					t.Fatal(err)
				}
				var member database.OrganizationMember
				if err := db.First(&member, "organization_id = ?", org.ID).Error; err != nil {
					t.Fatal(err)
				}
				if member.UserID != "new-user" || member.Role != auth.SystemRoleIDOwner || member.Status != "active" {
					t.Fatalf("owner=%+v", member)
				}
				var quota database.OrgQuota
				if err := db.First(&quota, "organization_id = ?", org.ID).Error; err != nil {
					t.Fatal(err)
				}
				if quota.PlanID != "plan-starter" {
					t.Fatalf("plan=%s", quota.PlanID)
				}
				var count int64
				db.Model(&database.Organization{}).Count(&count)
				if count != 1 {
					t.Fatalf("organizations=%d", count)
				}
			})
		}
	}
}
