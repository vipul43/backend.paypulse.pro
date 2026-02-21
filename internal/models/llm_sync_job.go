package models

import "time"

// LLM sync job status constants
const (
	LLMStatusPending    = "pending"
	LLMStatusProcessing = "processing"
	LLMStatusCompleted  = "completed"
	LLMStatusFailed     = "failed"
)

// LLMSyncJob represents a job for extracting payment information from an email using LLM
type LLMSyncJob struct {
	ID           string     `gorm:"column:id;primaryKey"`
	AccountID    string     `gorm:"column:accountId;index"`
	MessageID    string     `gorm:"column:messageId;uniqueIndex"`
	Status       string     `gorm:"column:status;index"`
	LastSyncedAt *time.Time `gorm:"column:lastSyncedAt"`
	Attempts     int        `gorm:"column:attempts"`
	LastError    *string    `gorm:"column:lastError"`
	CreatedAt    time.Time  `gorm:"column:createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updatedAt"`
	ProcessedAt  *time.Time `gorm:"column:processedAt"`
}

// TableName specifies the table name for GORM
func (LLMSyncJob) TableName() string {
	return "llm_sync_job"
}
