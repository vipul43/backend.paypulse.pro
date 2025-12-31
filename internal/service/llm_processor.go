package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/vipul43/kiwis-worker/internal/models"
	"github.com/vipul43/kiwis-worker/internal/openrouter"
	"github.com/vipul43/kiwis-worker/internal/repository"
)

const (
	LLMBatchSize      = 3    // Process 3 LLM jobs at a time (free models are very slow, ~30-60s per email, 3 emails = ~3-5 minutes)
	MaxEmailBodyChars = 5000 // Limit email body to 5000 characters for DDoS protection and efficient LLM processing
)

type LLMProcessor struct {
	accountRepo      *repository.AccountRepository
	llmSyncJobRepo   *repository.LLMSyncJobRepository
	paymentRepo      *repository.PaymentRepository
	gmailClient      GmailClient
	openRouterClient *openrouter.Client
}

func NewLLMProcessor(
	accountRepo *repository.AccountRepository,
	llmSyncJobRepo *repository.LLMSyncJobRepository,
	paymentRepo *repository.PaymentRepository,
	gmailClient GmailClient,
	openRouterClient *openrouter.Client,
) *LLMProcessor {
	return &LLMProcessor{
		accountRepo:      accountRepo,
		llmSyncJobRepo:   llmSyncJobRepo,
		paymentRepo:      paymentRepo,
		gmailClient:      gmailClient,
		openRouterClient: openRouterClient,
	}
}

// ProcessLLMSyncJobs processes a batch of LLM sync jobs
func (p *LLMProcessor) ProcessLLMSyncJobs(ctx context.Context, jobs []models.LLMSyncJob) error {
	if len(jobs) == 0 {
		return nil
	}

	log.Printf("Processing batch of %d LLM sync jobs", len(jobs))

	// Group jobs by account to get access token once per account
	jobsByAccount := make(map[string][]models.LLMSyncJob)
	for _, job := range jobs {
		jobsByAccount[job.AccountID] = append(jobsByAccount[job.AccountID], job)
	}

	// Process each account's jobs
	for accountID, accountJobs := range jobsByAccount {
		if err := p.processAccountJobs(ctx, accountID, accountJobs); err != nil {
			log.Printf("Error processing jobs for account %s: %v", accountID, err)
			// Continue with other accounts
		}
	}

	return nil
}

// processAccountJobs processes all jobs for a single account
func (p *LLMProcessor) processAccountJobs(ctx context.Context, accountID string, jobs []models.LLMSyncJob) error {
	// Fetch account details
	account, err := p.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		// Mark all jobs as failed
		for _, job := range jobs {
			errMsg := fmt.Sprintf("failed to get account: %v", err)
			_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusFailed, &errMsg)
		}
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Validate tokens exist
	if account.AccessToken == nil || account.RefreshToken == nil {
		errMsg := "account missing tokens"
		for _, job := range jobs {
			_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusFailed, &errMsg)
		}
		return fmt.Errorf("account missing tokens")
	}

	// Check if access token is expired and refresh if needed
	accessToken := *account.AccessToken
	if p.isTokenExpired(account.AccessTokenExpiresAt) {
		log.Printf("Access token expired for account %s, refreshing...", accountID)
		newToken, err := p.refreshToken(ctx, account)
		if err != nil {
			errMsg := fmt.Sprintf("failed to refresh token: %v", err)
			for _, job := range jobs {
				_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusFailed, &errMsg)
			}
			return fmt.Errorf("failed to refresh token: %w", err)
		}
		accessToken = newToken
	}

	// Fetch full emails for all message IDs
	log.Printf("Fetching %d emails for account %s", len(jobs), accountID)
	emails := make([]openrouter.EmailData, 0, len(jobs))
	jobIndexMap := make(map[int]models.LLMSyncJob) // Map email index to job

	for _, job := range jobs {
		email, err := p.fetchEmail(ctx, accessToken, job.MessageID)
		if err != nil {
			log.Printf("Failed to fetch email %s: %v", job.MessageID, err)
			errMsg := fmt.Sprintf("failed to fetch email: %v", err)
			_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusFailed, &errMsg)
			continue
		}
		emails = append(emails, *email)
		jobIndexMap[len(emails)-1] = job
	}

	if len(emails) == 0 {
		log.Printf("No emails to process for account %s", accountID)
		return nil
	}

	// Send batch to OpenRouter LLM
	log.Printf("Sending %d emails to LLM for payment extraction", len(emails))
	payments, rawResponses, err := p.openRouterClient.BatchExtractPayments(ctx, emails)
	if err != nil {
		// Mark all jobs as failed
		errMsg := fmt.Sprintf("LLM extraction failed: %v", err)
		for _, job := range jobs {
			_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusFailed, &errMsg)
		}
		return fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Process results
	paymentsToCreate := make([]models.Payment, 0)
	now := time.Now()

	for i, paymentData := range payments {
		job := jobIndexMap[i]
		rawResp := rawResponses[i]

		// Check if it's a valid payment
		if paymentData.Merchant == "" || paymentData.Amount == nil {
			// Not a payment email, mark job as completed
			log.Printf("Email %s is not a payment email, marking as completed", job.MessageID)
			_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusCompleted, nil)
			continue
		}

		// Parse date
		paymentDate, err := time.Parse(time.RFC3339, paymentData.Date)
		if err != nil {
			// Try alternative format without timezone
			paymentDate, err = time.Parse("2006-01-02T15:04:05", paymentData.Date)
			if err != nil {
				log.Printf("Failed to parse date %s: %v", paymentData.Date, err)
				errMsg := fmt.Sprintf("failed to parse date: %v", err)
				_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusFailed, &errMsg)
				continue
			}
		}

		// Create payment
		gmailURL := fmt.Sprintf("https://mail.google.com/mail/u/0/#inbox/%s", job.MessageID)
		payment := models.Payment{
			ID:                uuid.New().String(),
			AccountID:         accountID,
			Merchant:          paymentData.Merchant,
			Description:       paymentData.Description,
			Amount:            *paymentData.Amount,
			Currency:          paymentData.Currency,
			Date:              paymentDate,
			Recurrence:        paymentData.Recurrence,
			Status:            paymentData.Status,
			Category:          paymentData.Category,
			ExternalReference: &gmailURL,
			Metadata:          models.JSONB(paymentData.Metadata),
			RawLlmResponse:    models.JSONB(rawResp),
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		paymentsToCreate = append(paymentsToCreate, payment)

		// Mark job as completed
		_ = p.llmSyncJobRepo.UpdateStatus(ctx, job.ID, models.LLMStatusCompleted, nil)
		log.Printf("Extracted payment from email %s: %s - %.2f %s", job.MessageID, payment.Merchant, payment.Amount, payment.Currency)
	}

	// Bulk create payments
	if len(paymentsToCreate) > 0 {
		if err := p.paymentRepo.BulkCreate(ctx, paymentsToCreate); err != nil {
			return fmt.Errorf("failed to create payments: %w", err)
		}
		log.Printf("Created %d payments for account %s", len(paymentsToCreate), accountID)
	}

	return nil
}

// fetchEmail fetches a single email by message ID
func (p *LLMProcessor) fetchEmail(ctx context.Context, accessToken string, messageID string) (*openrouter.EmailData, error) {
	// Fetch email directly by ID
	msg, err := p.gmailClient.FetchEmailByID(ctx, accessToken, messageID)
	if err != nil {
		return nil, err
	}

	// Truncate body to prevent DDoS and reduce token usage
	// Payment info is typically in the first part of the email
	body := msg.BodyText
	if len(body) > MaxEmailBodyChars {
		body = body[:MaxEmailBodyChars]
	}

	return &openrouter.EmailData{
		From:    msg.From,
		Subject: msg.Subject,
		Body:    body,
	}, nil
}

// isTokenExpired checks if access token is expired or will expire within 5 minutes
func (p *LLMProcessor) isTokenExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return true // Assume expired if no expiry time
	}
	return time.Now().Add(5 * time.Minute).After(*expiresAt)
}

// refreshToken refreshes the access token and updates the account
func (p *LLMProcessor) refreshToken(ctx context.Context, account *models.Account) (string, error) {
	if account.RefreshToken == nil {
		return "", fmt.Errorf("no refresh token available")
	}

	result, err := p.gmailClient.RefreshAccessToken(ctx, *account.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update account with new tokens
	err = p.accountRepo.UpdateTokens(ctx, account.ID, result.AccessToken, result.RefreshToken, result.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to update tokens in database: %w", err)
	}

	log.Printf("Token refreshed for account %s, expires at %s", account.ID, result.ExpiresAt)

	return result.AccessToken, nil
}
