package repository

import (
	"context"
	"time"

	"github.com/vipul43/kiwis-worker/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LLMSyncJobRepository struct {
	db *gorm.DB
}

func NewLLMSyncJobRepository(db *gorm.DB) *LLMSyncJobRepository {
	return &LLMSyncJobRepository{db: db}
}

// Create creates a new LLM sync job
func (r *LLMSyncJobRepository) Create(ctx context.Context, job models.LLMSyncJob) error {
	return r.db.WithContext(ctx).Create(&job).Error
}

// BulkCreate creates multiple LLM sync jobs in a single transaction
// Uses ON CONFLICT DO NOTHING to skip duplicates by messageId
func (r *LLMSyncJobRepository) BulkCreate(ctx context.Context, jobs []models.LLMSyncJob) error {
	if len(jobs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "messageId"}},
			DoNothing: true,
		}).
		Create(&jobs).Error
}

// GetPendingJobs retrieves pending LLM sync jobs (round-robin by lastSyncedAt)
func (r *LLMSyncJobRepository) GetPendingJobs(ctx context.Context, limit int) ([]models.LLMSyncJob, error) {
	var jobs []models.LLMSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.LLMStatusPending).
		Order(`"lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	return jobs, result.Error
}

// GetFailedJobs retrieves failed LLM sync jobs for retry
func (r *LLMSyncJobRepository) GetFailedJobs(ctx context.Context, limit int) ([]models.LLMSyncJob, error) {
	var jobs []models.LLMSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.LLMStatusFailed).
		Order(`"lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	return jobs, result.Error
}

// GetProcessingJobs retrieves stuck processing jobs (crash recovery)
func (r *LLMSyncJobRepository) GetProcessingJobs(ctx context.Context, limit int) ([]models.LLMSyncJob, error) {
	var jobs []models.LLMSyncJob
	result := r.db.WithContext(ctx).
		Where("status = ?", models.LLMStatusProcessing).
		Order(`"lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC`).
		Limit(limit).
		Find(&jobs)
	return jobs, result.Error
}

// UpdateStatus updates the status of an LLM sync job
func (r *LLMSyncJobRepository) UpdateStatus(ctx context.Context, id string, status string, lastError *string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.LLMSyncJob{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       status,
			"lastError":    lastError,
			"updatedAt":    now,
			"lastSyncedAt": now,
		}).Error
}

// IncrementAttempts increments the attempts counter
func (r *LLMSyncJobRepository) IncrementAttempts(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&models.LLMSyncJob{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"attempts":  gorm.Expr("attempts + 1"),
			"updatedAt": time.Now(),
		}).Error
}
