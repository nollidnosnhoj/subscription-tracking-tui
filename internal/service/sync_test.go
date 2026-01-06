package service_test

import (
	"context"
	"testing"

	"subscription-tracker/internal/service"
)

func TestSyncService_ExportImportEncrypted(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	password := "test_password_123"

	// Create some test data
	_, err := tdb.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "Netflix",
		Amount:          15.99,
		Currency:        "USD",
		BillingCycle:    "monthly",
		NextRenewalDate: "2026-01-15",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	_, err = tdb.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "Amazon Prime",
		Amount:          139.00,
		Currency:        "USD",
		BillingCycle:    "yearly",
		NextRenewalDate: "2026-06-15",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	// Set some config
	if err := tdb.ConfigService.SetMonthCutoffDay(ctx, 22); err != nil {
		t.Fatalf("failed to set cutoff day: %v", err)
	}
	if err := tdb.ConfigService.SetMonthlySalary(ctx, 5000.00); err != nil {
		t.Fatalf("failed to set salary: %v", err)
	}

	// Export encrypted
	encrypted, err := tdb.SyncService.ExportEncrypted(ctx, password)
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}

	if encrypted == "" {
		t.Error("ExportEncrypted() returned empty string")
	}

	// Create a new test DB to import into
	tdb2 := setupTestDB(t)

	// Import encrypted
	if err := tdb2.SyncService.ImportEncrypted(ctx, encrypted, password); err != nil {
		t.Fatalf("ImportEncrypted() error = %v", err)
	}

	// Verify subscriptions were imported
	subs, err := tdb2.SubscriptionService.List(ctx, "")
	if err != nil {
		t.Fatalf("failed to list subscriptions: %v", err)
	}

	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}

	// Verify config was imported
	cutoff, err := tdb2.ConfigService.GetMonthCutoffDay(ctx)
	if err != nil {
		t.Fatalf("failed to get cutoff day: %v", err)
	}
	if cutoff != 22 {
		t.Errorf("cutoff day = %d, want 22", cutoff)
	}

	salary, err := tdb2.ConfigService.GetMonthlySalary(ctx)
	if err != nil {
		t.Fatalf("failed to get salary: %v", err)
	}
	if salary != 5000.00 {
		t.Errorf("salary = %.2f, want 5000.00", salary)
	}
}

func TestSyncService_ImportWrongPassword(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Create some test data
	_, err := tdb.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "Netflix",
		Amount:          15.99,
		Currency:        "USD",
		BillingCycle:    "monthly",
		NextRenewalDate: "2026-01-15",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	// Export with one password
	encrypted, err := tdb.SyncService.ExportEncrypted(ctx, "correct_password")
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}

	// Try to import with wrong password
	tdb2 := setupTestDB(t)
	err = tdb2.SyncService.ImportEncrypted(ctx, encrypted, "wrong_password")
	if err == nil {
		t.Error("ImportEncrypted() with wrong password should fail")
	}
}

func TestSyncService_ImportReplacesExistingData(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()
	password := "test_password"

	// Create initial data in source
	_, err := tdb.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "Netflix",
		Amount:          15.99,
		Currency:        "USD",
		BillingCycle:    "monthly",
		NextRenewalDate: "2026-01-15",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	// Export
	encrypted, err := tdb.SyncService.ExportEncrypted(ctx, password)
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}

	// Create different data in target
	tdb2 := setupTestDB(t)
	_, err = tdb2.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "Spotify",
		Amount:          9.99,
		Currency:        "USD",
		BillingCycle:    "monthly",
		NextRenewalDate: "2026-01-10",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
	_, err = tdb2.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "HBO Max",
		Amount:          14.99,
		Currency:        "USD",
		BillingCycle:    "monthly",
		NextRenewalDate: "2026-01-20",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	// Verify target has 2 subscriptions
	subs, _ := tdb2.SubscriptionService.List(ctx, "")
	if len(subs) != 2 {
		t.Fatalf("expected 2 subscriptions before import, got %d", len(subs))
	}

	// Import - should replace existing data
	if err := tdb2.SyncService.ImportEncrypted(ctx, encrypted, password); err != nil {
		t.Fatalf("ImportEncrypted() error = %v", err)
	}

	// Verify target now has 1 subscription (from source)
	subs, _ = tdb2.SubscriptionService.List(ctx, "")
	if len(subs) != 1 {
		t.Errorf("expected 1 subscription after import, got %d", len(subs))
	}

	if subs[0].Name != "Netflix" {
		t.Errorf("expected Netflix subscription, got %s", subs[0].Name)
	}
}
