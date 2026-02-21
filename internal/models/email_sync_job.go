package models

import "time"

type EmailSyncStatus string

const (
	EmailStatusPending    EmailSyncStatus = "pending"    // Ready to fetch next batch
	EmailStatusProcessing EmailSyncStatus = "processing" // Currently fetching
	EmailStatusSynced     EmailSyncStatus = "synced"     // All historical emails fetched, waiting for webhook
	EmailStatusCompleted  EmailSyncStatus = "completed"  // Webhook setup complete, job finished
	EmailStatusFailed     EmailSyncStatus = "failed"     // Failed after max retries
)

type EmailSyncType string

const (
	SyncTypeInitial     EmailSyncType = "initial"     // Initial historical sync
	SyncTypeIncremental EmailSyncType = "incremental" // Incremental sync (manual re-sync)
	SyncTypeWebhook     EmailSyncType = "webhook"     // Real-time sync (webhook-triggered)
)

type EmailSyncJob struct {
	ID            string          `gorm:"column:id;primaryKey"`
	AccountID     string          `gorm:"column:accountId;index"`
	Status        EmailSyncStatus `gorm:"column:status;index"`
	SyncType      EmailSyncType   `gorm:"column:syncType"`
	EmailsFetched int             `gorm:"column:emailsFetched"`
	PageToken     *string         `gorm:"column:pageToken"`
	LastSyncedAt  *time.Time      `gorm:"column:lastSyncedAt"`
	Attempts      int             `gorm:"column:attempts"`
	LastError     *string         `gorm:"column:lastError"`
	CreatedAt     time.Time       `gorm:"column:createdAt"`
	UpdatedAt     time.Time       `gorm:"column:updatedAt"`
	ProcessedAt   *time.Time      `gorm:"column:processedAt"`
}

// TableName specifies the table name for GORM
func (EmailSyncJob) TableName() string {
	return "email_sync_job"
}
