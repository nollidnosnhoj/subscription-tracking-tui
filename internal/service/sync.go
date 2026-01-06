package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"subscription-tracker/internal/db"
)

// SyncService handles encrypted backup and sync operations
type SyncService struct {
	queries       *db.Queries
	configService *ConfigService
}

// NewSyncService creates a new sync service
func NewSyncService(queries *db.Queries, configService *ConfigService) *SyncService {
	return &SyncService{
		queries:       queries,
		configService: configService,
	}
}

// SyncData represents all data to be synced
type SyncData struct {
	Version       int                `json:"version"`
	ExportedAt    time.Time          `json:"exported_at"`
	Subscriptions []SyncSubscription `json:"subscriptions"`
	Config        map[string]string  `json:"config"`
}

// SyncSubscription represents a subscription for sync
type SyncSubscription struct {
	Name            string  `json:"name"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	BillingCycle    string  `json:"billing_cycle"`
	NextRenewalDate string  `json:"next_renewal_date,omitempty"`
}

// ExportEncrypted exports all data as an encrypted string
func (s *SyncService) ExportEncrypted(ctx context.Context, password string) (string, error) {
	// Gather all data
	data, err := s.gatherData(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to gather data: %w", err)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encrypt
	encrypted, err := Encrypt(jsonData, password)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	return encrypted, nil
}

// ImportEncrypted imports data from an encrypted string
func (s *SyncService) ImportEncrypted(ctx context.Context, encrypted string, password string) error {
	// Decrypt
	jsonData, err := Decrypt(encrypted, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Parse JSON
	var data SyncData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to parse data: %w", err)
	}

	// Import data
	return s.importData(ctx, &data)
}

// gatherData collects all data for export
func (s *SyncService) gatherData(ctx context.Context) (*SyncData, error) {
	// Get subscriptions
	subs, err := s.queries.ListSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	syncSubs := make([]SyncSubscription, len(subs))
	for i, sub := range subs {
		syncSubs[i] = SyncSubscription{
			Name:         sub.Name,
			Amount:       sub.Amount,
			Currency:     sub.Currency,
			BillingCycle: sub.BillingCycle,
		}
		if sub.NextRenewalDate.Valid {
			syncSubs[i].NextRenewalDate = sub.NextRenewalDate.String
		}
	}

	// Get config
	configs, err := s.queries.GetAllConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	configMap := make(map[string]string)
	for _, c := range configs {
		configMap[c.Key] = c.Value
	}

	return &SyncData{
		Version:       1,
		ExportedAt:    time.Now().UTC(),
		Subscriptions: syncSubs,
		Config:        configMap,
	}, nil
}

// importData imports sync data into the database
func (s *SyncService) importData(ctx context.Context, data *SyncData) error {
	// Delete existing subscriptions
	subs, err := s.queries.ListSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list existing subscriptions: %w", err)
	}
	for _, sub := range subs {
		if err := s.queries.DeleteSubscription(ctx, sub.ID); err != nil {
			return fmt.Errorf("failed to delete subscription %s: %w", sub.Name, err)
		}
	}

	// Import subscriptions
	for _, sub := range data.Subscriptions {
		params := db.CreateSubscriptionParams{
			Name:         sub.Name,
			Amount:       sub.Amount,
			Currency:     sub.Currency,
			BillingCycle: sub.BillingCycle,
		}
		if sub.NextRenewalDate != "" {
			params.NextRenewalDate.String = sub.NextRenewalDate
			params.NextRenewalDate.Valid = true
		}
		if _, err := s.queries.CreateSubscription(ctx, params); err != nil {
			return fmt.Errorf("failed to create subscription %s: %w", sub.Name, err)
		}
	}

	// Import config
	for key, value := range data.Config {
		if err := s.queries.SetConfig(ctx, db.SetConfigParams{Key: key, Value: value}); err != nil {
			return fmt.Errorf("failed to set config %s: %w", key, err)
		}
	}

	return nil
}

// GitHub Gist API integration

const (
	gistAPIURL   = "https://api.github.com/gists"
	gistFileName = "subscription-tracker-backup.enc"
)

// GistConfig holds GitHub Gist configuration
type GistConfig struct {
	Token  string // GitHub personal access token
	GistID string // Gist ID (empty for new gist)
}

// PushToGist uploads encrypted data to a GitHub Gist
func (s *SyncService) PushToGist(ctx context.Context, password string, gistConfig GistConfig) (string, error) {
	// Export encrypted data
	encrypted, err := s.ExportEncrypted(ctx, password)
	if err != nil {
		return "", err
	}

	// Prepare gist payload
	payload := map[string]interface{}{
		"description": "Subscription Tracker Backup (encrypted)",
		"public":      false,
		"files": map[string]interface{}{
			gistFileName: map[string]string{
				"content": encrypted,
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gist payload: %w", err)
	}

	// Determine URL and method
	var url string
	var method string
	if gistConfig.GistID != "" {
		url = fmt.Sprintf("%s/%s", gistAPIURL, gistConfig.GistID)
		method = "PATCH"
	} else {
		url = gistAPIURL
		method = "POST"
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+gistConfig.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gist API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response to get gist ID
	var gistResp struct {
		ID      string `json:"id"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return "", fmt.Errorf("failed to parse gist response: %w", err)
	}

	return gistResp.ID, nil
}

// PullFromGist downloads and decrypts data from a GitHub Gist
func (s *SyncService) PullFromGist(ctx context.Context, password string, gistConfig GistConfig) error {
	if gistConfig.GistID == "" {
		return fmt.Errorf("gist ID is required for pull")
	}

	// Fetch gist
	url := fmt.Sprintf("%s/%s", gistAPIURL, gistConfig.GistID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+gistConfig.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gist API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var gistResp struct {
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return fmt.Errorf("failed to parse gist response: %w", err)
	}

	// Get encrypted content
	file, ok := gistResp.Files[gistFileName]
	if !ok {
		return fmt.Errorf("backup file not found in gist")
	}

	// Import encrypted data
	return s.ImportEncrypted(ctx, file.Content, password)
}

// Config keys for storing gist settings
const (
	ConfigKeyGistID    = "sync_gist_id"
	ConfigKeyGistToken = "sync_gist_token" // Note: storing tokens in DB isn't ideal, but encrypted
)

// GetGistConfig retrieves stored gist configuration
func (s *SyncService) GetGistConfig(ctx context.Context) (*GistConfig, error) {
	config := &GistConfig{}

	if token, err := s.queries.GetConfig(ctx, ConfigKeyGistToken); err == nil {
		config.Token = token
	}
	if gistID, err := s.queries.GetConfig(ctx, ConfigKeyGistID); err == nil {
		config.GistID = gistID
	}

	return config, nil
}

// SaveGistConfig saves gist configuration
func (s *SyncService) SaveGistConfig(ctx context.Context, config *GistConfig) error {
	if config.Token != "" {
		if err := s.queries.SetConfig(ctx, db.SetConfigParams{
			Key:   ConfigKeyGistToken,
			Value: config.Token,
		}); err != nil {
			return err
		}
	}
	if config.GistID != "" {
		if err := s.queries.SetConfig(ctx, db.SetConfigParams{
			Key:   ConfigKeyGistID,
			Value: config.GistID,
		}); err != nil {
			return err
		}
	}
	return nil
}
