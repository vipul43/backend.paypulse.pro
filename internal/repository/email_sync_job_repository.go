package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vipul43/kiwis-worker/internal/models"
	"gorm.io/gorm"
)

type EmailSyncJobRepository struct {
	db *gorm.DB
}

func NewEmailSyncJobRepository(db *gorm.DB) *EmailSyncJobRepository {
	return &EmailSyncJobRepository{db: db}
}

// GetPendingJobs retrieves pending email sync jobs in round-robin order
// New jobs (lastSyncedAt = NULL) get picked first, then oldest synced jobs
func (r *EmailSyncJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]models.EmailSyncJob, error) {
	var jobs []models.EmailSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.EmailStatusPending).
		Order(`"lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query pending jobs: %w", result.Error)
	}
	return jobs, nil
}

// GetFailedJobs retrieves failed email sync jobs for retry in round-robin order
func (r *EmailSyncJobRepository) GetFailedJobs(ctx context.Context, limit int) ([]models.EmailSyncJob, error) {
	var jobs []models.EmailSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.EmailStatusFailed).
		Order(`"lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query failed jobs: %w", result.Error)
	}
	return jobs, nil
}

// GetProcessingJobs retrieves email sync jobs stuck in processing state
func (r *EmailSyncJobRepository) GetProcessingJobs(ctx context.Context, limit int) ([]models.EmailSyncJob, error) {
	var jobs []models.EmailSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.EmailStatusProcessing).
		Order(`"lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to query processing jobs: %w", result.Error)
	}
	return jobs, nil
}

// Create creates a new email sync job
func (r *EmailSyncJobRepository) Create(ctx context.Context, job models.EmailSyncJob) error {
	result := r.db.WithContext(ctx).Create(&job)
	if result.Error != nil {
		return fmt.Errorf("failed to create email sync job: %w", result.Error)
	}
	return nil
}

// UpdateProgress updates job progress (emails fetched, page token, last synced time)
// Updates lastSyncedAt to push job to back of round-robin queue
func (r *EmailSyncJobRepository) UpdateProgress(ctx context.Context, jobID string, emailsFetched int, pageToken *string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.EmailSyncJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"emailsFetched": emailsFetched,
			"pageToken":     pageToken,
			"lastSyncedAt":  now,
			"updatedAt":     now,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to update job progress: %w", result.Error)
	}
	return nil
}

// UpdateStatus updates the job status
// For synced/completed/failed status, sets processedAt
// For processing status, clears processedAt
func (r *EmailSyncJobRepository) UpdateStatus(ctx context.Context, jobID string, status models.EmailSyncStatus, lastError *string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":    status,
		"lastError": lastError,
		"updatedAt": now,
	}

	// Set processedAt for terminal states (synced, completed, failed)
	// Clear it for processing state (job is being worked on)
	if status == models.EmailStatusSynced || status == models.EmailStatusCompleted || status == models.EmailStatusFailed {
		updates["processedAt"] = &now
	}

	result := r.db.WithContext(ctx).Model(&models.EmailSyncJob{}).
		Where("id = ?", jobID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update job status: %w", result.Error)
	}
	return nil
}

// IncrementAttempts increments the retry attempt counter
func (r *EmailSyncJobRepository) IncrementAttempts(ctx context.Context, jobID string) error {
	result := r.db.WithContext(ctx).Model(&models.EmailSyncJob{}).
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

// GetByID retrieves an email sync job by ID
func (r *EmailSyncJobRepository) GetByID(ctx context.Context, jobID string) (*models.EmailSyncJob, error) {
	var job models.EmailSyncJob
	result := r.db.WithContext(ctx).First(&job, "id = ?", jobID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", result.Error)
	}
	return &job, nil
}
