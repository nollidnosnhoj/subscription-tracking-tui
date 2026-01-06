package service_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"subscription-tracker/internal/service"
)

func TestExportService_Export_CSV(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	inputs := []service.CreateSubscriptionInput{
		{Name: "Netflix", Amount: 15.99, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2026-01-15"},
		{Name: "Spotify", Amount: 9.99, Currency: "EUR", BillingCycle: "monthly", NextRenewalDate: "2026-01-20"},
		{Name: "Amazon Prime", Amount: 139.00, Currency: "USD", BillingCycle: "yearly", NextRenewalDate: "2026-06-15"},
	}

	for _, input := range inputs {
		if _, err := tdb.SubscriptionService.Create(ctx, input); err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}
	}

	var buf bytes.Buffer
	count, err := tdb.ExportService.Export(ctx, &buf, service.FormatCSV)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if count != 3 {
		t.Errorf("Export() count = %d, want 3", count)
	}

	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	if len(records) != 4 {
		t.Errorf("expected 4 rows (1 header + 3 data), got %d", len(records))
	}

	// Verify header
	expectedHeader := []string{"ID", "Name", "Amount", "Currency", "Billing Cycle", "Next Renewal Date", "Created At", "Updated At"}
	for i, h := range expectedHeader {
		if records[0][i] != h {
			t.Errorf("header[%d] = %s, want %s", i, records[0][i], h)
		}
	}
}

func TestExportService_Export_JSON(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	inputs := []service.CreateSubscriptionInput{
		{Name: "Netflix", Amount: 15.99, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2026-01-15"},
		{Name: "Spotify", Amount: 9.99, Currency: "EUR", BillingCycle: "monthly", NextRenewalDate: "2026-01-20"},
	}

	for _, input := range inputs {
		if _, err := tdb.SubscriptionService.Create(ctx, input); err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}
	}

	var buf bytes.Buffer
	count, err := tdb.ExportService.Export(ctx, &buf, service.FormatJSON)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if count != 2 {
		t.Errorf("Export() count = %d, want 2", count)
	}

	var exported []service.ExportSubscription
	if err := json.Unmarshal(buf.Bytes(), &exported); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(exported) != 2 {
		t.Errorf("expected 2 subscriptions in JSON, got %d", len(exported))
	}
}

func TestExportService_Export_EmptyDatabase(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	var buf bytes.Buffer
	count, err := tdb.ExportService.Export(ctx, &buf, service.FormatCSV)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if count != 0 {
		t.Errorf("Export() count = %d, want 0", count)
	}
}

func TestExportService_Export_InvalidFormat(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	_, err := tdb.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "Test",
		Amount:          10.00,
		Currency:        "USD",
		BillingCycle:    "monthly",
		NextRenewalDate: "2026-01-15",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	var buf bytes.Buffer
	_, err = tdb.ExportService.Export(ctx, &buf, "invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}
