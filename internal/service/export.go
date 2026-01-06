package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"subscription-tracker/internal/db"
)

// ExportService handles export functionality
type ExportService struct {
	queries *db.Queries
}

// NewExportService creates a new export service
func NewExportService(queries *db.Queries) *ExportService {
	return &ExportService{queries: queries}
}

// ExportFormat represents the export format
type ExportFormat string

const (
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportSubscription represents a subscription for export
type ExportSubscription struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	BillingCycle    string  `json:"billing_cycle"`
	NextRenewalDate string  `json:"next_renewal_date,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// Export exports subscriptions to the given writer in the specified format
func (s *ExportService) Export(ctx context.Context, w io.Writer, format ExportFormat) (int, error) {
	subs, err := s.queries.GetAllSubscriptionsForExport(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subs) == 0 {
		return 0, nil
	}

	switch format {
	case FormatCSV:
		return len(subs), s.exportCSV(w, subs)
	case FormatJSON:
		return len(subs), s.exportJSON(w, subs)
	default:
		return 0, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *ExportService) exportCSV(w io.Writer, subs []db.Subscription) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	header := []string{"ID", "Name", "Amount", "Currency", "Billing Cycle", "Next Renewal Date", "Created At", "Updated At"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Rows
	for _, sub := range subs {
		renewalDate := ""
		if sub.NextRenewalDate.Valid {
			renewalDate = sub.NextRenewalDate.String
		}

		row := []string{
			fmt.Sprintf("%d", sub.ID),
			sub.Name,
			fmt.Sprintf("%.2f", sub.Amount),
			sub.Currency,
			sub.BillingCycle,
			renewalDate,
			sub.CreatedAt,
			sub.UpdatedAt,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

func (s *ExportService) exportJSON(w io.Writer, subs []db.Subscription) error {
	var exportData []ExportSubscription

	for _, sub := range subs {
		renewalDate := ""
		if sub.NextRenewalDate.Valid {
			renewalDate = sub.NextRenewalDate.String
		}

		exportData = append(exportData, ExportSubscription{
			ID:              sub.ID,
			Name:            sub.Name,
			Amount:          sub.Amount,
			Currency:        sub.Currency,
			BillingCycle:    sub.BillingCycle,
			NextRenewalDate: renewalDate,
			CreatedAt:       sub.CreatedAt,
			UpdatedAt:       sub.UpdatedAt,
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(exportData)
}

// ConvertToExportFormat converts db subscriptions to export format
func ConvertToExportFormat(subs []db.Subscription) []ExportSubscription {
	result := make([]ExportSubscription, len(subs))
	for i, sub := range subs {
		renewalDate := ""
		if sub.NextRenewalDate.Valid {
			renewalDate = sub.NextRenewalDate.String
		}

		result[i] = ExportSubscription{
			ID:              sub.ID,
			Name:            sub.Name,
			Amount:          sub.Amount,
			Currency:        sub.Currency,
			BillingCycle:    sub.BillingCycle,
			NextRenewalDate: renewalDate,
			CreatedAt:       sub.CreatedAt,
			UpdatedAt:       sub.UpdatedAt,
		}
	}
	return result
}
