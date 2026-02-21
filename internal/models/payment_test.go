package models

import (
	"testing"
	"time"
)

func TestPaymentStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"draft", PaymentStatusDraft, "draft"},
		{"scheduled", PaymentStatusScheduled, "scheduled"},
		{"unpaid", PaymentStatusUnpaid, "unpaid"},
		{"processing", PaymentStatusProcessing, "processing"},
		{"partially_paid", PaymentStatusPartiallyPaid, "partially_paid"},
		{"paid", PaymentStatusPaid, "paid"},
		{"failed", PaymentStatusFailed, "failed"},
		{"refunded", PaymentStatusRefunded, "refunded"},
		{"cancelled", PaymentStatusCancelled, "cancelled"},
		{"written_off", PaymentStatusWrittenOff, "written_off"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

func TestPaymentRecurrence_Constants(t *testing.T) {
	tests := []struct {
		name       string
		recurrence string
		expected   string
	}{
		{"daily", RecurrenceDaily, "daily"},
		{"weekly", RecurrenceWeekly, "weekly"},
		{"biweekly", RecurrenceBiweekly, "biweekly"},
		{"monthly", RecurrenceMonthly, "monthly"},
		{"bimonthly", RecurrenceBimonthly, "bimonthly"},
		{"quarterly", RecurrenceQuarterly, "quarterly"},
		{"semiannual", RecurrenceSemiannual, "semiannual"},
		{"annual", RecurrenceAnnual, "annual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.recurrence != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.recurrence)
			}
		})
	}
}

func TestPayment_Structure(t *testing.T) {
	now := time.Now()
	description := "Test payment"
	category := "subscription"
	externalRef := "https://mail.google.com/mail/u/0/#inbox/abc123"
	recurrence := RecurrenceMonthly

	payment := Payment{
		ID:                "payment-123",
		AccountID:         "account-456",
		Merchant:          "Netflix",
		Description:       &description,
		Amount:            19.99,
		Currency:          "USD",
		Date:              now,
		Recurrence:        &recurrence,
		Status:            PaymentStatusUnpaid,
		Category:          &category,
		ExternalReference: &externalRef,
		Metadata:          JSONB{"plan": "premium"},
		RawLlmResponse:    JSONB{"raw": "data"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if payment.ID != "payment-123" {
		t.Errorf("Expected ID 'payment-123', got %s", payment.ID)
	}
	if payment.Merchant != "Netflix" {
		t.Errorf("Expected Merchant 'Netflix', got %s", payment.Merchant)
	}
	if payment.Amount != 19.99 {
		t.Errorf("Expected Amount 19.99, got %f", payment.Amount)
	}
	if payment.Status != PaymentStatusUnpaid {
		t.Errorf("Expected Status 'unpaid', got %s", payment.Status)
	}
}
