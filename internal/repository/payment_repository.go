package repository

import (
	"context"

	"github.com/vipul43/kiwis-worker/internal/models"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create creates a new payment
func (r *PaymentRepository) Create(ctx context.Context, payment models.Payment) error {
	return r.db.WithContext(ctx).Create(&payment).Error
}

// BulkCreate creates multiple payments in a single transaction
func (r *PaymentRepository) BulkCreate(ctx context.Context, payments []models.Payment) error {
	if len(payments) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&payments).Error
}

// GetByAccountID retrieves all payments for an account
func (r *PaymentRepository) GetByAccountID(ctx context.Context, accountID string) ([]models.Payment, error) {
	var payments []models.Payment
	result := r.db.WithContext(ctx).
		Where(`"accountId" = ?`, accountID).
		Order("date DESC").
		Find(&payments)
	return payments, result.Error
}
