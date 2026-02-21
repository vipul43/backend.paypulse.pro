package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Payment status constants
const (
	PaymentStatusDraft         = "draft"
	PaymentStatusScheduled     = "scheduled"
	PaymentStatusUpcoming      = "upcoming"
	PaymentStatusDue           = "due"
	PaymentStatusOverdue       = "overdue"
	PaymentStatusProcessing    = "processing"
	PaymentStatusPartiallyPaid = "partially_paid"
	PaymentStatusPaid          = "paid"
	PaymentStatusFailed        = "failed"
	PaymentStatusRefunded      = "refunded"
	PaymentStatusCancelled     = "cancelled"
	PaymentStatusWrittenOff    = "written_off"
)

// Payment recurrence constants
const (
	RecurrenceDaily      = "daily"
	RecurrenceWeekly     = "weekly"
	RecurrenceBiweekly   = "biweekly"
	RecurrenceMonthly    = "monthly"
	RecurrenceBimonthly  = "bimonthly"
	RecurrenceQuarterly  = "quarterly"
	RecurrenceSemiannual = "semiannual"
	RecurrenceAnnual     = "annual"
)

// JSONB type for GORM to handle PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements driver.Valuer for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// Payment represents a payment extracted from an email
type Payment struct {
	ID                string    `gorm:"column:id;primaryKey"`
	AccountID         string    `gorm:"column:accountId;index"`
	Merchant          string    `gorm:"column:merchant;index"`
	Description       *string   `gorm:"column:description"`
	Amount            float64   `gorm:"column:amount"`
	Currency          string    `gorm:"column:currency"`
	Date              time.Time `gorm:"column:date;index"`
	Recurrence        *string   `gorm:"column:recurrence"`
	Status            string    `gorm:"column:status;index"`
	Category          *string   `gorm:"column:category"`
	ExternalReference *string   `gorm:"column:externalReference"`
	Metadata          JSONB     `gorm:"column:metadata;type:jsonb"`
	RawLlmResponse    JSONB     `gorm:"column:rawLlmResponse;type:jsonb"`
	CreatedAt         time.Time `gorm:"column:createdAt;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updatedAt;autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Payment) TableName() string {
	return "payment"
}
