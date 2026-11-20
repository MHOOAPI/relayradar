package store

import (
	"context"
	"testing"
	"time"
)

func TestMemorySubscriptions(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	now := time.Now().UTC()

	sub := Subscription{
		ID: "sub-test", URL: "https://example.com/hook", Secret: "s3cret",
		Kinds: []string{"price_change"}, Stations: []string{"aihubmix"},
		CreatedAt: now, LastSentAt: now,
	}
	if err := m.CreateSubscription(ctx, sub); err != nil {
		t.Fatal(err)
	}

	list, err := m.ListSubscriptions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "sub-test" || len(list[0].Kinds) != 1 {
		t.Fatalf("unexpected list: %+v", list)
	}

	later := now.Add(time.Minute)
	if err := m.TouchSubscription(ctx, "sub-test", later); err != nil {
		t.Fatal(err)
	}
	list, _ = m.ListSubscriptions(ctx)
	if !list[0].LastSentAt.Equal(later) {
		t.Fatalf("last sent not updated: %v", list[0].LastSentAt)
	}

	if err := m.DeleteSubscription(ctx, "sub-test"); err != nil {
		t.Fatal(err)
	}
	list, _ = m.ListSubscriptions(ctx)
	if len(list) != 0 {
		t.Fatalf("expected empty, got %d", len(list))
	}
}
