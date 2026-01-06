package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"subscription-tracker/internal/db"
)

// SubscriptionService handles subscription business logic
type SubscriptionService struct {
	queries *db.Queries
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(queries *db.Queries) *SubscriptionService {
	return &SubscriptionService{queries: queries}
}

// CreateSubscriptionInput represents input for creating a subscription
type CreateSubscriptionInput struct {
	Name            string
	Amount          float64
	Currency        string
	BillingCycle    string // "monthly" or "yearly"
	NextRenewalDate string // YYYY-MM-DD format, required for yearly, optional for monthly (defaults to 1st)
}

// Validate validates the input
func (i *CreateSubscriptionInput) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("name is required")
	}
	if i.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if i.Currency == "" {
		i.Currency = "USD"
	}
	if i.BillingCycle != "monthly" && i.BillingCycle != "yearly" {
		return fmt.Errorf("billing cycle must be 'monthly' or 'yearly'")
	}
	// Renewal date is required for all subscriptions
	if i.NextRenewalDate == "" {
		return fmt.Errorf("renewal date is required")
	}
	if _, err := time.Parse("2006-01-02", i.NextRenewalDate); err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}
	return nil
}

// Create creates a new subscription
func (s *SubscriptionService) Create(ctx context.Context, input CreateSubscriptionInput) (db.Subscription, error) {
	if err := input.Validate(); err != nil {
		return db.Subscription{}, err
	}

	params := db.CreateSubscriptionParams{
		Name:            input.Name,
		Amount:          input.Amount,
		Currency:        input.Currency,
		BillingCycle:    input.BillingCycle,
		NextRenewalDate: sql.NullString{String: input.NextRenewalDate, Valid: true},
	}

	return s.queries.CreateSubscription(ctx, params)
}

// Get retrieves a subscription by ID
func (s *SubscriptionService) Get(ctx context.Context, id int64) (db.Subscription, error) {
	return s.queries.GetSubscription(ctx, id)
}

// List retrieves all subscriptions, optionally filtered by billing cycle
func (s *SubscriptionService) List(ctx context.Context, billingCycle string) ([]db.Subscription, error) {
	if billingCycle != "" {
		return s.queries.ListSubscriptionsByBillingCycle(ctx, billingCycle)
	}
	return s.queries.ListSubscriptions(ctx)
}

// ListMonthly retrieves all monthly subscriptions
func (s *SubscriptionService) ListMonthly(ctx context.Context) ([]db.Subscription, error) {
	return s.queries.ListMonthlySubscriptions(ctx)
}

// ListYearly retrieves all yearly subscriptions
func (s *SubscriptionService) ListYearly(ctx context.Context) ([]db.Subscription, error) {
	return s.queries.ListYearlySubscriptions(ctx)
}

// UpdateSubscriptionInput represents input for updating a subscription
type UpdateSubscriptionInput struct {
	ID              int64
	Name            string
	Amount          float64
	Currency        string
	BillingCycle    string
	NextRenewalDate string // Required for yearly, optional for monthly
}

// Validate validates the update input
func (i *UpdateSubscriptionInput) Validate() error {
	if i.ID <= 0 {
		return fmt.Errorf("invalid subscription ID")
	}
	if i.Name == "" {
		return fmt.Errorf("name is required")
	}
	if i.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if i.BillingCycle != "monthly" && i.BillingCycle != "yearly" {
		return fmt.Errorf("billing cycle must be 'monthly' or 'yearly'")
	}
	// Renewal date is required for all subscriptions
	if i.NextRenewalDate == "" {
		return fmt.Errorf("renewal date is required")
	}
	if _, err := time.Parse("2006-01-02", i.NextRenewalDate); err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}
	return nil
}

// Update updates an existing subscription
func (s *SubscriptionService) Update(ctx context.Context, input UpdateSubscriptionInput) (db.Subscription, error) {
	if err := input.Validate(); err != nil {
		return db.Subscription{}, err
	}

	params := db.UpdateSubscriptionParams{
		ID:              input.ID,
		Name:            input.Name,
		Amount:          input.Amount,
		Currency:        input.Currency,
		BillingCycle:    input.BillingCycle,
		NextRenewalDate: sql.NullString{String: input.NextRenewalDate, Valid: true},
	}

	return s.queries.UpdateSubscription(ctx, params)
}

// UpdateRenewalDate updates only the renewal date (for yearly subscriptions)
func (s *SubscriptionService) UpdateRenewalDate(ctx context.Context, id int64, newDate string) (db.Subscription, error) {
	if _, err := time.Parse("2006-01-02", newDate); err != nil {
		return db.Subscription{}, fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	return s.queries.UpdateRenewalDate(ctx, db.UpdateRenewalDateParams{
		ID:              id,
		NextRenewalDate: sql.NullString{String: newDate, Valid: true},
	})
}

// Delete removes a subscription
func (s *SubscriptionService) Delete(ctx context.Context, id int64) error {
	return s.queries.DeleteSubscription(ctx, id)
}

// AdvanceRenewalDates checks all subscriptions and advances their renewal dates
// if they are in the past. Monthly subscriptions advance by 1 month, yearly by 1 year.
func (s *SubscriptionService) AdvanceRenewalDates(ctx context.Context) error {
	return s.AdvanceRenewalDatesFrom(ctx, time.Now())
}

// AdvanceRenewalDatesFrom advances renewal dates that are before the given reference time.
// This is useful for testing with a specific date.
func (s *SubscriptionService) AdvanceRenewalDatesFrom(ctx context.Context, referenceTime time.Time) error {
	subs, err := s.queries.ListSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	today := time.Date(referenceTime.Year(), referenceTime.Month(), referenceTime.Day(), 0, 0, 0, 0, time.UTC)

	for _, sub := range subs {
		if !sub.NextRenewalDate.Valid {
			continue
		}

		renewalDate, err := time.Parse("2006-01-02", sub.NextRenewalDate.String)
		if err != nil {
			continue
		}

		// If renewal date is in the past, advance it
		if renewalDate.Before(today) {
			newDate := CalculateNextRenewalDate(renewalDate, sub.BillingCycle, today)
			_, err := s.queries.UpdateRenewalDate(ctx, db.UpdateRenewalDateParams{
				ID:              sub.ID,
				NextRenewalDate: sql.NullString{String: newDate.Format("2006-01-02"), Valid: true},
			})
			if err != nil {
				return fmt.Errorf("failed to update renewal date for %s: %w", sub.Name, err)
			}
		}
	}

	return nil
}

// CalculateNextRenewalDate calculates the next renewal date after the reference time.
// For monthly subscriptions, it advances by months keeping the same day.
// For yearly subscriptions, it advances by years keeping the same month and day.
func CalculateNextRenewalDate(currentRenewal time.Time, billingCycle string, referenceTime time.Time) time.Time {
	newDate := currentRenewal

	if billingCycle == "monthly" {
		// Advance by months until we're at or after the reference time
		for newDate.Before(referenceTime) {
			newDate = addMonth(newDate)
		}
	} else {
		// Yearly: advance by years
		for newDate.Before(referenceTime) {
			newDate = newDate.AddDate(1, 0, 0)
		}
	}

	return newDate
}

// addMonth adds one month to the date, handling edge cases like Jan 31 -> Feb 28
func addMonth(t time.Time) time.Time {
	year, month, day := t.Year(), t.Month(), t.Day()

	// Move to next month
	month++
	if month > 12 {
		month = 1
		year++
	}

	// Handle edge cases where day doesn't exist in next month (e.g., Jan 31 -> Feb)
	// Find the last day of the new month
	lastDayOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDayOfMonth {
		day = lastDayOfMonth
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
