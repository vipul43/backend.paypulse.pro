package models

import "time"

type AccountSyncStatus string

const (
	StatusPending    AccountSyncStatus = "pending"
	StatusProcessing AccountSyncStatus = "processing"
	StatusCompleted  AccountSyncStatus = "completed"
	StatusFailed     AccountSyncStatus = "failed"
)

type AccountSyncJob struct {
	ID          string            `gorm:"column:id;primaryKey"`
	AccountID   string            `gorm:"column:accountId;uniqueIndex"`
	Status      AccountSyncStatus `gorm:"column:status"`
	Attempts    int               `gorm:"column:attempts"`
	LastError   *string           `gorm:"column:lastError"`
	CreatedAt   time.Time         `gorm:"column:createdAt"`
	UpdatedAt   time.Time         `gorm:"column:updatedAt"`
	ProcessedAt *time.Time        `gorm:"column:processedAt"`
}

// TableName specifies the table name for GORM
func (AccountSyncJob) TableName() string {
	return "account_sync_job"
}
