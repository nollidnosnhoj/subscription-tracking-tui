package service_test

import (
	"context"
	"testing"

	"subscription-tracker/internal/service"
)

func TestSpendingService_CalculateForMonth(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Create test subscriptions
	// Monthly subs: Netflix renews on the 15th, Spotify on the 20th
	monthlyInputs := []service.CreateSubscriptionInput{
		{Name: "Netflix", Amount: 15.99, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2026-01-15"},
		{Name: "Spotify", Amount: 9.99, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2026-01-20"},
	}

	yearlyInputs := []service.CreateSubscriptionInput{
		{Name: "Amazon Prime", Amount: 139.00, Currency: "USD", BillingCycle: "yearly", NextRenewalDate: "2026-01-10"},
		{Name: "Adobe CC", Amount: 599.88, Currency: "USD", BillingCycle: "yearly", NextRenewalDate: "2026-06-15"},
	}

	for _, input := range monthlyInputs {
		if _, err := tdb.SubscriptionService.Create(ctx, input); err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}
	}
	for _, input := range yearlyInputs {
		if _, err := tdb.SubscriptionService.Create(ctx, input); err != nil {
			t.Fatalf("failed to create subscription: %v", err)
		}
	}

	// With default cutoff day = 1:
	// - "February 2026" period = Jan 1 to Jan 31
	//   - Monthly: Netflix (15th) and Spotify (20th) both renew in this period
	//   - Yearly: Amazon Prime (Jan 10) renews in this period
	// - "July 2026" period = Jun 1 to Jun 30
	//   - Monthly: Netflix (15th) and Spotify (20th) both renew in this period
	//   - Yearly: Adobe CC (Jun 15) renews in this period
	// - "April 2026" period = Mar 1 to Mar 31
	//   - Monthly: Netflix (15th) and Spotify (20th) both renew in this period
	//   - Yearly: no yearly renewals
	tests := []struct {
		name                 string
		year                 int
		month                int
		expectedMonthlySum   float64
		expectedMonthlyCount int
		expectedYearlyCount  int
	}{
		{
			name:                 "February 2026 - has Amazon Prime yearly renewal",
			year:                 2026,
			month:                2,
			expectedMonthlySum:   25.98, // Netflix + Spotify
			expectedMonthlyCount: 2,
			expectedYearlyCount:  1, // Amazon Prime renews Jan 10
		},
		{
			name:                 "July 2026 - has Adobe CC yearly renewal",
			year:                 2026,
			month:                7,
			expectedMonthlySum:   25.98,
			expectedMonthlyCount: 2,
			expectedYearlyCount:  1, // Adobe CC renews Jun 15
		},
		{
			name:                 "April 2026 - no yearly renewals",
			year:                 2026,
			month:                4,
			expectedMonthlySum:   25.98,
			expectedMonthlyCount: 2,
			expectedYearlyCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := tdb.SpendingService.CalculateForMonth(ctx, tt.year, tt.month)
			if err != nil {
				t.Fatalf("CalculateForMonth() error = %v", err)
			}

			if len(summary.MonthlyItems) != tt.expectedMonthlyCount {
				t.Errorf("monthly count = %d, want %d", len(summary.MonthlyItems), tt.expectedMonthlyCount)
			}
			if len(summary.YearlyItems) != tt.expectedYearlyCount {
				t.Errorf("yearly count = %d, want %d", len(summary.YearlyItems), tt.expectedYearlyCount)
			}
			if !almostEqual(summary.MonthlyTotal, tt.expectedMonthlySum) {
				t.Errorf("MonthlyTotal = %.2f, want %.2f", summary.MonthlyTotal, tt.expectedMonthlySum)
			}
		})
	}
}

func TestSpendingService_CalculateForMonth_WithCustomCutoff(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Set cutoff day to 22
	if err := tdb.ConfigService.SetMonthCutoffDay(ctx, 22); err != nil {
		t.Fatalf("failed to set cutoff day: %v", err)
	}

	// Create a yearly subscription that renews on Jan 5, 2026
	_, err := tdb.SubscriptionService.Create(ctx, service.CreateSubscriptionInput{
		Name:            "Test Yearly",
		Amount:          100.00,
		Currency:        "USD",
		BillingCycle:    "yearly",
		NextRenewalDate: "2026-01-05",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	// With cutoff = 22:
	// - "January 2026" period = Dec 22, 2025 to Jan 21, 2026
	// - Jan 5 falls within this period, so it should be included
	summary, err := tdb.SpendingService.CalculateForMonth(ctx, 2026, 1)
	if err != nil {
		t.Fatalf("CalculateForMonth() error = %v", err)
	}

	// Verify the period dates
	expectedStart := "2025-12-22"
	expectedEnd := "2026-01-21" // End of Jan 21 (23:59:59)
	actualStart := summary.PeriodStart.Format("2006-01-02")
	actualEnd := summary.PeriodEnd.Format("2006-01-02")

	if actualStart != expectedStart {
		t.Errorf("PeriodStart = %s, want %s", actualStart, expectedStart)
	}
	if actualEnd != expectedEnd {
		t.Errorf("PeriodEnd = %s, want %s", actualEnd, expectedEnd)
	}

	// The yearly subscription should be included (Jan 5 is in Dec 22 - Jan 21)
	if len(summary.YearlyItems) != 1 {
		t.Errorf("yearly count = %d, want 1", len(summary.YearlyItems))
	}
}

func TestSpendingService_InvalidMonth(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		month int
	}{
		{"month 0", 0},
		{"month 13", 13},
		{"month -1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tdb.SpendingService.CalculateForMonth(ctx, 2026, tt.month)
			if err == nil {
				t.Error("expected error for invalid month")
			}
		})
	}
}

func TestConfigService_MonthlySalary(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Default should be 0
	salary, err := tdb.ConfigService.GetMonthlySalary(ctx)
	if err != nil {
		t.Fatalf("GetMonthlySalary() error = %v", err)
	}
	if salary != 0 {
		t.Errorf("default salary = %.2f, want 0", salary)
	}

	// Set salary
	err = tdb.ConfigService.SetMonthlySalary(ctx, 5000.00)
	if err != nil {
		t.Fatalf("SetMonthlySalary() error = %v", err)
	}

	salary, err = tdb.ConfigService.GetMonthlySalary(ctx)
	if err != nil {
		t.Fatalf("GetMonthlySalary() error = %v", err)
	}
	if salary != 5000.00 {
		t.Errorf("salary = %.2f, want 5000.00", salary)
	}

	// Negative salary should fail
	err = tdb.ConfigService.SetMonthlySalary(ctx, -100)
	if err == nil {
		t.Error("expected error for negative salary")
	}
}

func TestSpendingService_RemainingMoney(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Set salary to 3000
	if err := tdb.ConfigService.SetMonthlySalary(ctx, 3000.00); err != nil {
		t.Fatalf("failed to set salary: %v", err)
	}

	// Create subscriptions - both renew on the 15th so they fall in Jan period
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
		Name:            "Spotify",
		Amount:          9.99,
		Currency:        "USD",
		BillingCycle:    "monthly",
		NextRenewalDate: "2026-01-15",
	})
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	// February period = Jan 1 to Jan 31, both subs renew on 15th
	summary, err := tdb.SpendingService.CalculateForMonth(ctx, 2026, 2)
	if err != nil {
		t.Fatalf("CalculateForMonth() error = %v", err)
	}

	if summary.MonthlySalary != 3000.00 {
		t.Errorf("MonthlySalary = %.2f, want 3000.00", summary.MonthlySalary)
	}

	expectedRemaining := 3000.00 - 25.98 // salary - (Netflix + Spotify)
	if !almostEqual(summary.Remaining, expectedRemaining) {
		t.Errorf("Remaining = %.2f, want %.2f", summary.Remaining, expectedRemaining)
	}
}

func TestConfigService_MonthCutoffDay(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Default should be 1
	day, err := tdb.ConfigService.GetMonthCutoffDay(ctx)
	if err != nil {
		t.Fatalf("GetMonthCutoffDay() error = %v", err)
	}
	if day != 1 {
		t.Errorf("default cutoff day = %d, want 1", day)
	}

	// Set to 22
	err = tdb.ConfigService.SetMonthCutoffDay(ctx, 22)
	if err != nil {
		t.Fatalf("SetMonthCutoffDay() error = %v", err)
	}

	day, err = tdb.ConfigService.GetMonthCutoffDay(ctx)
	if err != nil {
		t.Fatalf("GetMonthCutoffDay() error = %v", err)
	}
	if day != 22 {
		t.Errorf("cutoff day = %d, want 22", day)
	}

	// Invalid day should fail
	err = tdb.ConfigService.SetMonthCutoffDay(ctx, 29)
	if err == nil {
		t.Error("expected error for day > 28")
	}

	err = tdb.ConfigService.SetMonthCutoffDay(ctx, 0)
	if err == nil {
		t.Error("expected error for day < 1")
	}
}

func TestSpendingService_MonthlyRenewalInPeriod(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Test that monthly subscriptions are counted when their renewal DAY
	// falls within any month of the billing period.
	// Monthly subscriptions recur every month on the same day.

	// Set cutoff day to 15
	if err := tdb.ConfigService.SetMonthCutoffDay(ctx, 15); err != nil {
		t.Fatalf("failed to set cutoff day: %v", err)
	}

	// Create monthly subscriptions with different renewal days
	// Cutoff 15 means:
	// - "February 2026" period = Jan 15 to Feb 14
	//
	// For monthly subscriptions, we check if the renewal DAY falls in the period:
	// - Day 20: Jan 20 is in [Jan 15, Feb 14] → included
	// - Day 10: Feb 10 is in [Jan 15, Feb 14] → included
	// - Day 14: Feb 14 is in [Jan 15, Feb 14] → included (boundary)
	// - Day 15: Jan 15 is in [Jan 15, Feb 14] → included (boundary)
	//
	// Actually, every day 1-28 will have at least one occurrence in a ~30 day period!
	// So all monthly subscriptions should be included.

	inputs := []service.CreateSubscriptionInput{
		{Name: "Sub Day 20", Amount: 10.00, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2026-01-20"},
		{Name: "Sub Day 10", Amount: 10.00, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2026-02-10"},
	}

	for _, input := range inputs {
		_, err := tdb.SubscriptionService.Create(ctx, input)
		if err != nil {
			t.Fatalf("failed to create subscription %s: %v", input.Name, err)
		}
	}

	// Calculate for February 2026 (period = Jan 15 to Feb 14)
	summary, err := tdb.SpendingService.CalculateForMonth(ctx, 2026, 2)
	if err != nil {
		t.Fatalf("CalculateForMonth() error = %v", err)
	}

	// All monthly subscriptions should be included (their day occurs in the ~30 day period)
	if len(summary.MonthlyItems) != 2 {
		t.Errorf("monthly count = %d, want 2", len(summary.MonthlyItems))
		for _, item := range summary.MonthlyItems {
			t.Logf("  - %s (renewal: %s)", item.Name, item.NextRenewalDate.String)
		}
	}

	// Monthly total should be sum of all
	expectedTotal := 20.00
	if !almostEqual(summary.MonthlyTotal, expectedTotal) {
		t.Errorf("MonthlyTotal = %.2f, want %.2f", summary.MonthlyTotal, expectedTotal)
	}
}

func TestSpendingService_MonthlyRenewalCrossingYearBoundary(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	// Test monthly renewal detection when period crosses year boundary
	// Cutoff 20 means "January 2026" period = Dec 20, 2025 to Jan 19, 2026

	if err := tdb.ConfigService.SetMonthCutoffDay(ctx, 20); err != nil {
		t.Fatalf("failed to set cutoff day: %v", err)
	}

	// Create subscriptions with different renewal days
	inputs := []service.CreateSubscriptionInput{
		{Name: "Sub Day 25", Amount: 10.00, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2025-12-25"},
		{Name: "Sub Day 10", Amount: 20.00, Currency: "USD", BillingCycle: "monthly", NextRenewalDate: "2026-01-10"},
	}

	for _, input := range inputs {
		_, err := tdb.SubscriptionService.Create(ctx, input)
		if err != nil {
			t.Fatalf("failed to create subscription %s: %v", input.Name, err)
		}
	}

	// Calculate for January 2026 (period = Dec 20, 2025 to Jan 19, 2026)
	summary, err := tdb.SpendingService.CalculateForMonth(ctx, 2026, 1)
	if err != nil {
		t.Fatalf("CalculateForMonth() error = %v", err)
	}

	// Both monthly subscriptions should be included
	if len(summary.MonthlyItems) != 2 {
		t.Errorf("monthly count = %d, want 2", len(summary.MonthlyItems))
		for _, item := range summary.MonthlyItems {
			t.Logf("  - %s (renewal: %s)", item.Name, item.NextRenewalDate.String)
		}
	}

	// Verify period dates cross year boundary correctly
	if summary.PeriodStart.Year() != 2025 || summary.PeriodStart.Month() != 12 || summary.PeriodStart.Day() != 20 {
		t.Errorf("PeriodStart = %s, want 2025-12-20", summary.PeriodStart.Format("2006-01-02"))
	}
	if summary.PeriodEnd.Year() != 2026 || summary.PeriodEnd.Month() != 1 || summary.PeriodEnd.Day() != 19 {
		t.Errorf("PeriodEnd = %s, want 2026-01-19", summary.PeriodEnd.Format("2006-01-02"))
	}

	// Monthly total should include all
	expectedTotal := 30.00
	if !almostEqual(summary.MonthlyTotal, expectedTotal) {
		t.Errorf("MonthlyTotal = %.2f, want %.2f", summary.MonthlyTotal, expectedTotal)
	}
}

func TestParseMonth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{"number 1", "1", 1, false},
		{"number 6", "6", 6, false},
		{"number 12", "12", 12, false},
		{"full name January", "January", 1, false},
		{"full name December", "December", 12, false},
		{"short name Jan", "Jan", 1, false},
		{"number 0 invalid", "0", 0, true},
		{"number 13 invalid", "13", 0, true},
		{"invalid string", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ParseMonth(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMonth(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMonth(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseMonth(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}
