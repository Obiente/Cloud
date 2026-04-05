package gameservers

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestResolveGameServerLogOptions(t *testing.T) {
	defaultLimit := int32(100)
	search := "  Error  "
	since := timestamppb.New(time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC))

	limit, sinceTime, untilTime, searchQuery := resolveGameServerLogOptions(nil, since, nil, &search, defaultLimit)
	if limit != defaultLimit {
		t.Fatalf("expected default limit %d, got %d", defaultLimit, limit)
	}
	if sinceTime != nil {
		t.Fatalf("expected since time to be cleared when converting to historical until query")
	}
	if untilTime == nil || !untilTime.Equal(since.AsTime()) {
		t.Fatalf("expected until time to be derived from since timestamp")
	}
	if searchQuery != "error" {
		t.Fatalf("expected normalized search query, got %q", searchQuery)
	}
}

func TestResolveGameServerLogOptionsKeepsExplicitUntil(t *testing.T) {
	limitValue := int32(250)
	search := "Info"
	since := timestamppb.New(time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC))
	until := timestamppb.New(time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC))

	limit, sinceTime, untilTime, searchQuery := resolveGameServerLogOptions(&limitValue, since, until, &search, 100)
	if limit != limitValue {
		t.Fatalf("expected explicit limit %d, got %d", limitValue, limit)
	}
	if sinceTime == nil || !sinceTime.Equal(since.AsTime()) {
		t.Fatalf("expected since time to be preserved when until is provided")
	}
	if untilTime == nil || !untilTime.Equal(until.AsTime()) {
		t.Fatalf("expected until time to be preserved")
	}
	if searchQuery != "info" {
		t.Fatalf("expected normalized search query, got %q", searchQuery)
	}
}
