package support

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	supportv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/support/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSupportServiceUserIsolation(t *testing.T) {
	db := newSupportServiceTestDB(t)
	service := NewService(db)
	seedSupportServiceIsolationData(t, db)

	ctx := auth.WithUser(context.Background(), &authv1.User{
		Id:    "user-org-a",
		Email: "user-org-a@example.com",
	})

	listOrgA, err := service.ListTickets(ctx, connect.NewRequest(&supportv1.ListTicketsRequest{
		OrganizationId: proto.String("org-a"),
	}))
	if err != nil {
		t.Fatalf("list org-a tickets: %v", err)
	}
	if got := ticketIDs(listOrgA.Msg.Tickets); !slices.Equal(got, []string{"ticket-org-a-owner"}) {
		t.Fatalf("org-a list returned %v, want only user's org-a ticket", got)
	}

	listOrgB, err := service.ListTickets(ctx, connect.NewRequest(&supportv1.ListTicketsRequest{
		OrganizationId: proto.String("org-b"),
	}))
	if err != nil {
		t.Fatalf("list org-b tickets: %v", err)
	}
	if got := ticketIDs(listOrgB.Msg.Tickets); len(got) != 0 {
		t.Fatalf("org-b list returned %v for org-a user, want none", got)
	}

	_, err = service.GetTicket(ctx, connect.NewRequest(&supportv1.GetTicketRequest{
		TicketId: "ticket-org-b-owner",
	}))
	if err == nil {
		t.Fatal("cross-user ticket get succeeded, want permission error")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-user ticket get code = %v, want %v: %v", connect.CodeOf(err), connect.CodePermissionDenied, err)
	}
}

func newSupportServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&database.SupportTicket{}, &database.TicketComment{}); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}

	previousDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = previousDB
	})

	return db
}

func seedSupportServiceIsolationData(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Now().UTC()
	orgA := "org-a"
	orgB := "org-b"
	tickets := []*database.SupportTicket{
		testSupportTicket("ticket-org-a-owner", "Org A ticket", "user-org-a", &orgA, now),
		testSupportTicket("ticket-org-a-peer", "Org A peer ticket", "user-org-a-peer", &orgA, now),
		testSupportTicket("ticket-org-b-owner", "Org B ticket", "user-org-b", &orgB, now),
	}
	for _, ticket := range tickets {
		if err := db.Create(ticket).Error; err != nil {
			t.Fatalf("seed ticket %s: %v", ticket.ID, err)
		}
	}
}

func testSupportTicket(id, subject, createdBy string, orgID *string, now time.Time) *database.SupportTicket {
	return &database.SupportTicket{
		ID:             id,
		Subject:        subject,
		Description:    subject + " description",
		Status:         int32(supportv1.SupportTicketStatus_OPEN),
		Priority:       int32(supportv1.SupportTicketPriority_MEDIUM),
		Category:       int32(supportv1.SupportTicketCategory_TECHNICAL),
		CreatedBy:      createdBy,
		OrganizationID: orgID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func ticketIDs(tickets []*supportv1.SupportTicket) []string {
	ids := make([]string, 0, len(tickets))
	for _, ticket := range tickets {
		ids = append(ids, ticket.GetId())
	}
	slices.Sort(ids)
	return ids
}
