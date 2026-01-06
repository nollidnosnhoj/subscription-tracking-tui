package service

import (
	"context"
	"fmt"
	"strconv"

	"subscription-tracker/internal/db"
)

const (
	ConfigKeyMonthCutoffDay = "month_cutoff_day"
	ConfigKeyMonthlySalary  = "monthly_salary"
)

// ConfigService handles configuration
type ConfigService struct {
	queries *db.Queries
}

// NewConfigService creates a new config service
func NewConfigService(queries *db.Queries) *ConfigService {
	return &ConfigService{queries: queries}
}

// GetMonthCutoffDay returns the day of month when a new billing period starts
// Default is 1 (first of month)
func (s *ConfigService) GetMonthCutoffDay(ctx context.Context) (int, error) {
	value, err := s.queries.GetConfig(ctx, ConfigKeyMonthCutoffDay)
	if err != nil {
		// Return default if not found
		return 1, nil
	}

	day, err := strconv.Atoi(value)
	if err != nil {
		return 1, nil
	}

	if day < 1 || day > 28 {
		return 1, nil
	}

	return day, nil
}

// SetMonthCutoffDay sets the day of month when a new billing period starts
func (s *ConfigService) SetMonthCutoffDay(ctx context.Context, day int) error {
	if day < 1 || day > 28 {
		return fmt.Errorf("cutoff day must be between 1 and 28")
	}

	return s.queries.SetConfig(ctx, db.SetConfigParams{
		Key:   ConfigKeyMonthCutoffDay,
		Value: strconv.Itoa(day),
	})
}

// GetMonthlySalary returns the user's monthly salary (pay stub amount)
// Returns 0 if not set
func (s *ConfigService) GetMonthlySalary(ctx context.Context) (float64, error) {
	value, err := s.queries.GetConfig(ctx, ConfigKeyMonthlySalary)
	if err != nil {
		return 0, nil
	}

	salary, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, nil
	}

	return salary, nil
}

// SetMonthlySalary sets the user's monthly salary (pay stub amount)
func (s *ConfigService) SetMonthlySalary(ctx context.Context, salary float64) error {
	if salary < 0 {
		return fmt.Errorf("salary cannot be negative")
	}

	return s.queries.SetConfig(ctx, db.SetConfigParams{
		Key:   ConfigKeyMonthlySalary,
		Value: strconv.FormatFloat(salary, 'f', 2, 64),
	})
}

// Config represents the application configuration
type Config struct {
	MonthCutoffDay int
	MonthlySalary  float64
}

// GetAll returns all configuration values
func (s *ConfigService) GetAll(ctx context.Context) (*Config, error) {
	cutoffDay, err := s.GetMonthCutoffDay(ctx)
	if err != nil {
		return nil, err
	}

	salary, err := s.GetMonthlySalary(ctx)
	if err != nil {
		return nil, err
	}

	return &Config{
		MonthCutoffDay: cutoffDay,
		MonthlySalary:  salary,
	}, nil
}
