package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"subscription-tracker/internal/db"
)

// SpendingService handles spending calculation logic
type SpendingService struct {
	queries       *db.Queries
	configService *ConfigService
}

// NewSpendingService creates a new spending service
func NewSpendingService(queries *db.Queries, configService *ConfigService) *SpendingService {
	return &SpendingService{
		queries:       queries,
		configService: configService,
	}
}

// SpendingSummary represents spending for a given billing period
type SpendingSummary struct {
	Year           int
	Month          int
	CutoffDay      int
	PeriodStart    time.Time
	PeriodEnd      time.Time
	MonthlyTotal   float64
	YearlyTotal    float64
	GrandTotal     float64
	MonthlyItems   []db.Subscription
	YearlyItems    []db.Subscription
	AverageMonthly float64 // Monthly + (Yearly / 12)
	MonthlySalary  float64 // User's monthly salary from config
	Remaining      float64 // Salary - GrandTotal (0 if no salary set)
}

// CalculateForMonth calculates spending for a specific billing period
// The period starts on cutoffDay of the previous month and ends on cutoffDay-1 of the given month
// Example: January with cutoff 22 = Dec 22 to Jan 21
func (s *SpendingService) CalculateForMonth(ctx context.Context, year, month int) (*SpendingSummary, error) {
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("month must be between 1 and 12")
	}

	cutoffDay, err := s.configService.GetMonthCutoffDay(ctx)
	if err != nil {
		cutoffDay = 1
	}

	// Calculate period start: cutoffDay of the previous month
	prevMonth := month - 1
	prevYear := year
	if prevMonth < 1 {
		prevMonth = 12
		prevYear--
	}
	periodStart := time.Date(prevYear, time.Month(prevMonth), cutoffDay, 0, 0, 0, 0, time.UTC)

	// Calculate period end: day before cutoffDay of the current month (end of that day)
	periodEnd := time.Date(year, time.Month(month), cutoffDay, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	// Get monthly subscriptions that renew during this period
	monthlySubs, err := s.getMonthlySubscriptionsInPeriod(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly subscriptions: %w", err)
	}

	// Get yearly subscriptions that renew during this period
	yearlySubs, err := s.getYearlySubscriptionsInPeriod(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get yearly subscriptions: %w", err)
	}

	summary := &SpendingSummary{
		Year:         year,
		Month:        month,
		CutoffDay:    cutoffDay,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		MonthlyItems: monthlySubs,
		YearlyItems:  yearlySubs,
	}

	// Calculate totals
	for _, sub := range monthlySubs {
		summary.MonthlyTotal += sub.Amount
	}
	for _, sub := range yearlySubs {
		summary.YearlyTotal += sub.Amount
	}

	summary.GrandTotal = summary.MonthlyTotal + summary.YearlyTotal
	summary.AverageMonthly = summary.MonthlyTotal + (summary.YearlyTotal / 12)

	// Get salary and calculate remaining
	salary, err := s.configService.GetMonthlySalary(ctx)
	if err == nil && salary > 0 {
		summary.MonthlySalary = salary
		summary.Remaining = salary - summary.GrandTotal
	}

	return summary, nil
}

// getYearlySubscriptionsInPeriod returns yearly subscriptions that renew within the given period
func (s *SpendingService) getYearlySubscriptionsInPeriod(ctx context.Context, start, end time.Time) ([]db.Subscription, error) {
	yearlySubs, err := s.queries.ListYearlySubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	var result []db.Subscription
	for _, sub := range yearlySubs {
		if !sub.NextRenewalDate.Valid {
			continue
		}

		renewalDate, err := time.Parse("2006-01-02", sub.NextRenewalDate.String)
		if err != nil {
			continue
		}

		// Check if renewal falls within the period
		if isDateInPeriod(renewalDate, start, end) {
			result = append(result, sub)
		}
	}

	return result, nil
}

// getMonthlySubscriptionsInPeriod returns monthly subscriptions that renew within the given period.
// A monthly subscription renews on the same day each month. We check if the stored renewal date
// falls within the period, OR if a future occurrence of that day falls within the period.
func (s *SpendingService) getMonthlySubscriptionsInPeriod(ctx context.Context, start, end time.Time) ([]db.Subscription, error) {
	monthlySubs, err := s.queries.ListMonthlySubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	var result []db.Subscription
	for _, sub := range monthlySubs {
		if !sub.NextRenewalDate.Valid {
			continue
		}

		renewalDate, err := time.Parse("2006-01-02", sub.NextRenewalDate.String)
		if err != nil {
			continue
		}

		// Check if the stored renewal date itself falls in the period
		if isDateInPeriod(renewalDate, start, end) {
			result = append(result, sub)
			continue
		}

		// For monthly subscriptions, also check if the renewal day would occur in this period
		// This handles cases where the stored date is in a different month but the day recurs
		renewalInPeriod := calculateMonthlyRenewalInPeriod(renewalDate.Day(), start, end)
		if renewalInPeriod != nil {
			result = append(result, sub)
		}
	}

	return result, nil
}

// calculateMonthlyRenewalInPeriod determines if a monthly subscription with a given renewal day
// would renew within the specified period. Returns the renewal date if it falls in the period, nil otherwise.
func calculateMonthlyRenewalInPeriod(renewalDay int, periodStart, periodEnd time.Time) *time.Time {
	// Check each month that overlaps with the period
	// Start from the month of periodStart
	current := time.Date(periodStart.Year(), periodStart.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(periodEnd.Year(), periodEnd.Month(), 1, 0, 0, 0, 0, time.UTC)

	for !current.After(endMonth) {
		// Calculate the renewal date for this month, handling edge cases
		lastDayOfMonth := time.Date(current.Year(), current.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		day := renewalDay
		if day > lastDayOfMonth {
			day = lastDayOfMonth
		}

		renewalDate := time.Date(current.Year(), current.Month(), day, 0, 0, 0, 0, time.UTC)

		// Check if this renewal date falls within the period
		if isDateInPeriod(renewalDate, periodStart, periodEnd) {
			return &renewalDate
		}

		// Move to next month
		current = current.AddDate(0, 1, 0)
	}

	return nil
}

// isDateInPeriod checks if a date falls within [start, end] (inclusive)
func isDateInPeriod(date, start, end time.Time) bool {
	return (date.Equal(start) || date.After(start)) && (date.Before(end) || date.Equal(end))
}

// CalculateForCurrentMonth calculates spending for the current billing period
func (s *SpendingService) CalculateForCurrentMonth(ctx context.Context) (*SpendingSummary, error) {
	now := time.Now()
	cutoffDay, _ := s.configService.GetMonthCutoffDay(ctx)

	// Determine which period we're in
	// Period for month M runs from cutoffDay of M-1 to cutoffDay-1 of M
	// If today >= cutoffDay, we're in next month's period
	year, month := now.Year(), int(now.Month())
	if now.Day() >= cutoffDay {
		// We're past the cutoff, so we're in the next month's period
		month++
		if month > 12 {
			month = 1
			year++
		}
	}

	return s.CalculateForMonth(ctx, year, month)
}

// CalculateAnnualTotal calculates total annual spending
func (s *SpendingService) CalculateAnnualTotal(ctx context.Context) (float64, error) {
	subs, err := s.queries.ListSubscriptions(ctx)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, sub := range subs {
		if sub.BillingCycle == "monthly" {
			total += sub.Amount * 12
		} else {
			total += sub.Amount
		}
	}

	return total, nil
}

// ParseMonth parses a month string (number or name) to an int
func ParseMonth(monthStr string) (int, error) {
	if monthStr == "" {
		return int(time.Now().Month()), nil
	}

	// Try parsing as number
	var month int
	if _, err := fmt.Sscanf(monthStr, "%d", &month); err == nil {
		if month < 1 || month > 12 {
			return 0, fmt.Errorf("month must be between 1 and 12, got: %d", month)
		}
		return month, nil
	}

	// Try parsing as month name
	for i := time.January; i <= time.December; i++ {
		if monthStr == i.String() || monthStr == i.String()[:3] {
			return int(i), nil
		}
	}

	return 0, fmt.Errorf("invalid month: %s (use 1-12 or month name)", monthStr)
}

// GetYearlySubscriptionsRenewingInMonth is a helper for getting yearly subs for a specific YYYY-MM
func (s *SpendingService) GetYearlySubscriptionsRenewingInMonth(ctx context.Context, yearMonth string) ([]db.Subscription, error) {
	return s.queries.GetYearlySubscriptionsRenewingInMonth(ctx, sql.NullString{String: yearMonth, Valid: true})
}
