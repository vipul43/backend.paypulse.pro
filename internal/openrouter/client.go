package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	OpenRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
	model      *string // Optional: if nil, uses OpenRouter account default
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // 5 minutes timeout for LLM calls (free models are slow)
		},
		model: nil, // Use OpenRouter account default
	}
}

// SetModel sets a specific model to use (optional)
func (c *Client) SetModel(model string) {
	c.model = &model
}

// EmailData represents the email data to extract payment from
type EmailData struct {
	From    string
	Subject string
	Body    string
}

// PaymentData represents the extracted payment information
type PaymentData struct {
	Merchant    string                 `json:"merchant"`
	Description *string                `json:"description"`
	Amount      *float64               `json:"amount"`
	Currency    string                 `json:"currency"`
	Date        string                 `json:"date"`
	Recurrence  *string                `json:"recurrence"`
	Status      string                 `json:"status"`
	Category    *string                `json:"category"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// BatchExtractPayments extracts payment information from multiple emails using OpenRouter batch API
func (c *Client) BatchExtractPayments(ctx context.Context, emails []EmailData) ([]PaymentData, []map[string]interface{}, error) {
	if len(emails) == 0 {
		return nil, nil, nil
	}

	// For now, process sequentially (OpenRouter free tier may not support true batching)
	// TODO: Implement true batch API when available
	results := make([]PaymentData, 0, len(emails))
	rawResponses := make([]map[string]interface{}, 0, len(emails))

	for _, email := range emails {
		payment, rawResp, err := c.ExtractPayment(ctx, email)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to extract payment: %w", err)
		}

		// Only add if it's a valid payment (has required fields)
		if payment != nil {
			results = append(results, *payment)
			rawResponses = append(rawResponses, rawResp)
		} else {
			// Not a payment email, add nil placeholder
			results = append(results, PaymentData{})
			rawResponses = append(rawResponses, rawResp)
		}
	}

	return results, rawResponses, nil
}

// ExtractPayment extracts payment information from a single email
func (c *Client) ExtractPayment(ctx context.Context, email EmailData) (*PaymentData, map[string]interface{}, error) {
	prompt := c.buildPrompt(email)

	reqBody := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	// Only include model if explicitly set, otherwise use OpenRouter account default
	if c.model != nil {
		reqBody["model"] = *c.model
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", OpenRouterAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse OpenRouter response
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("no response from LLM")
	}

	content := apiResp.Choices[0].Message.Content

	// Store raw response for audit
	var rawResponse map[string]interface{}
	_ = json.Unmarshal(body, &rawResponse)

	// Clean the content (remove markdown code blocks if present)
	cleanedContent := c.cleanJSONResponse(content)

	// Check if LLM returned null (low confidence or not a payment email)
	if cleanedContent == "null" || cleanedContent == "" {
		return nil, rawResponse, nil
	}

	// Parse payment data from LLM response
	var paymentData PaymentData
	if err := json.Unmarshal([]byte(cleanedContent), &paymentData); err != nil {
		return nil, rawResponse, fmt.Errorf("failed to parse payment JSON: %w", err)
	}

	// Validate required fields
	if !c.isValidPayment(paymentData) {
		return nil, rawResponse, nil
	}

	return &paymentData, rawResponse, nil
}

// cleanJSONResponse removes markdown code blocks and extra whitespace from LLM response
func (c *Client) cleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)

	// Check for null response first
	if content == "null" {
		return "null"
	}

	// Find the first { and last } to extract just the JSON object
	startIdx := strings.Index(content, "{")
	endIdx := strings.LastIndex(content, "}")

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		// No valid JSON found, return as is and let JSON parser fail with proper error
		return content
	}

	// Extract just the JSON object
	jsonContent := content[startIdx : endIdx+1]

	return strings.TrimSpace(jsonContent)
}

// buildPrompt builds the LLM prompt from email data
func (c *Client) buildPrompt(email EmailData) string {
	return fmt.Sprintf(`You are an AI that extracts structured payment information from emails.

Analyze the input and return a JSON object for the payment table. Only return the JSON if you are ≥75%% confident in ALL required fields. Otherwise, return: null

### OUTPUT FORMAT
{
  "merchant": "",
  "description": null,
  "amount": null,
  "currency": "",
  "date": null,
  "recurrence": null,
  "status": "",
  "category": null,
  "metadata": {}
}

### REQUIRED FIELDS (if any cannot be inferred with ≥75%% confidence → return null)

| Field | Type | Rules |
|-------|------|-------|
| merchant | string | Business/entity name exactly as it appears in input |
| amount | number | Total due amount. Numeric only, no symbols/commas. Positive always (even for refunds). Breakups go to metadata |
| currency | string | ISO 4217 code (INR, USD, EUR, GBP, JPY, etc.) |
| date | string | ISO 8601 with timezone (YYYY-MM-DDTHH:MM:SS±HH:MM). The due date, payment date, or transaction date from the email |
| status | string | One of: draft, scheduled, unpaid, processing, partially_paid, paid, failed, refunded, cancelled, written_off. Use 'unpaid' for pending payments (bills, invoices, dues) |

### OPTIONAL FIELDS (null if not inferable)

| Field | Type | Rules |
|-------|------|-------|
| description | string | What the payment is for |
| recurrence | string | daily, weekly, biweekly, monthly, bimonthly, quarterly, semiannual, annual. Infer from context if not explicit |
| category | string | subscription, utility, emi, credit_card_bill, loan, insurance, rent, misc. credit_card_bill is ONLY for credit card dues/statements, not payments made via credit card |
| metadata | object | Flat JSON with all additional inferred details: invoice_number, subscription_id, order_id, utr, reference_number, card_last_four, billing_period, plan_name, minimum_due, payment_method, etc. Use {} if none |

### STATUS GUIDE
- unpaid: Bills, invoices, dues that need to be paid (most common for payment reminder emails)
- paid: Payment confirmation, receipt, successful transaction
- failed: Payment failed, declined, unsuccessful
- refunded: Refund processed
- cancelled: Payment/subscription cancelled
- processing: Payment is being processed
- partially_paid: Partial payment made
- scheduled: Payment scheduled for future
- draft: Invoice in draft state
- written_off: Debt written off

### RULES
- Return ONLY raw JSON or null. No explanations, no markdown.
- All values must be inferred from input. Never fabricate.
- Merchant name should be preserved exactly as found, no normalization.
- Promotional/marketing emails (e.g., "Pay now and get X") → return null
- Confidence < 75%% on any required field → return null

### INPUT
from: %s
subject: %s
body: %s`, email.From, email.Subject, email.Body)
}

// isValidPayment checks if the payment data has all required fields
func (c *Client) isValidPayment(payment PaymentData) bool {
	// Required fields: merchant, amount, currency, date, status
	if payment.Merchant == "" {
		return false
	}
	if payment.Amount == nil || *payment.Amount <= 0 {
		return false
	}
	if payment.Currency == "" {
		return false
	}
	if payment.Date == "" {
		return false
	}
	if payment.Status == "" {
		return false
	}
	return true
}
