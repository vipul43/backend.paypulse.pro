package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/vipul43/kiwis-worker/internal/models"
	"gorm.io/gorm"
)

type AccountSyncJobRepository struct {
	db *gorm.DB
}

func NewAccountSyncJobRepository(db *gorm.DB) *AccountSyncJobRepository {
	return &AccountSyncJobRepository{db: db}
}

// CreateMissingJobsForAccountsOlderThan creates pending account sync jobs for accounts
// older than cutoff that don't have one yet.
// Uses ON CONFLICT DO NOTHING to stay safe under concurrent inserts/trigger execution.
func (r *AccountSyncJobRepository) CreateMissingJobsForAccountsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO account_sync_job (id, "accountId", status, "createdAt", "updatedAt")
		SELECT
			gen_random_uuid()::text,
			a.id,
			?,
			NOW(),
			NOW()
		FROM account a
		LEFT JOIN account_sync_job asj ON asj."accountId" = a.id
		WHERE asj.id IS NULL
		  AND a."createdAt" <= ?
		ON CONFLICT ("accountId") DO NOTHING
	`, models.StatusPending, cutoff)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to create missing account sync jobs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// GetPendingJobs retrieves all pending account sync jobs
func (r *AccountSyncJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]models.AccountSyncJob, error) {
	var jobs []models.AccountSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.StatusPending).
		Order(`"createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query pending jobs: %w", result.Error)
	}
	return jobs, nil
}

// GetFailedJobs retrieves all failed account sync jobs for retry
func (r *AccountSyncJobRepository) GetFailedJobs(ctx context.Context, limit int) ([]models.AccountSyncJob, error) {
	var jobs []models.AccountSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.StatusFailed).
		Order(`"createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query failed jobs: %w", result.Error)
	}
	return jobs, nil
}

// GetProcessingJobs retrieves account sync jobs stuck in processing state
func (r *AccountSyncJobRepository) GetProcessingJobs(ctx context.Context, limit int) ([]models.AccountSyncJob, error) {
	var jobs []models.AccountSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.StatusProcessing).
		Order(`"createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query processing jobs: %w", result.Error)
	}
	return jobs, nil
}

// UpdateStatus updates the job status
func (r *AccountSyncJobRepository) UpdateStatus(ctx context.Context, jobID string, status models.AccountSyncStatus, lastError *string) error {
	updates := map[string]interface{}{
		"status":    status,
		"lastError": lastError,
		"updatedAt": time.Now(),
	}

	if status == models.StatusCompleted || status == models.StatusFailed {
		now := time.Now()
		updates["processedAt"] = &now
	}

	result := r.db.WithContext(ctx).Model(&models.AccountSyncJob{}).
		Where("id = ?", jobID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update job status: %w", result.Error)
	}
	return nil
}

// IncrementAttempts increments the retry attempt counter
func (r *AccountSyncJobRepository) IncrementAttempts(ctx context.Context, jobID string) error {
	result := r.db.WithContext(ctx).Model(&models.AccountSyncJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"attempts":  gorm.Expr("attempts + 1"),
			"updatedAt": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to increment attempts: %w", result.Error)
	}
	return nil
}
