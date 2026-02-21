# Kiwis Worker

An AI-powered app that securely extracts and displays your upcoming payment details from Gmail without storing your email data.

This repository contains the backend worker service that watches for new OAuth accounts and triggers email processing jobs.

## How It Works

1. Frontend inserts new Account (OAuth flow)
2. PostgreSQL trigger automatically creates AccountSyncJob (status: pending)
3. Go watcher polls for pending jobs every 10 seconds
4. Processes account (placeholder for Gmail API integration)
5. Updates job status to completed/failed

**Account Deletion**: When an account is deleted, the sync job is automatically deleted via CASCADE foreign key constraint.

## Quick Start

```bash
# Install dependencies
go mod download

# Install migration CLI
make migrate-install

# Setup database (creates tables with camelCase columns)
psql "$DATABASE_URL" -f test_setup.sql

# Run migrations
make migrate-up

# Start the service
make run
```

The service will:
- Connect to PostgreSQL
- Run pending migrations (creates `account_sync_job` table and trigger)
- Process any pending jobs from previous runs
- Start polling for new accounts

## Project Structure

```
.
├── cmd/watcher/              # Application entry point
├── internal/
│   ├── config/              # Configuration
│   ├── database/            # Connection & migrations
│   ├── models/              # Data structures (type-safe enums)
│   ├── repository/          # Data access layer
│   ├── service/             # Business logic
│   └── watcher/             # Polling & orchestration
├── migrations/              # SQL migrations
└── test_setup.sql          # Test database setup
```

## Configuration

Edit `.env`:
- `DATABASE_URL`: PostgreSQL connection string (required)
- `GOOGLE_CLIENT_ID`: Google OAuth client ID (required for Gmail API)
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret (required for Gmail API)
- `OPENROUTER_API_KEY`: OpenRouter API key (for payment extraction)

Example:
```
DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"
GOOGLE_CLIENT_ID="123456-abc.apps.googleusercontent.com"
GOOGLE_CLIENT_SECRET="GOCSPX-xyz123"
OPENROUTER_API_KEY="sk-or-v1-..."
```

Defaults (in code):
- Poll interval: 10 seconds
- Infinite retries (no max limit)
- Shutdown timeout: 30 seconds
- Email batch size: 50 emails per fetch
- Max emails per account: 10,000
- Historical sync: 1 year of emails
- LLM batch size: 3 emails per batch
- Email body limit: 5,000 characters (DDoS protection)

## Database Schema

### Account Table (camelCase columns - Prisma/frontend)
- `id`, `accountId`, `providerId`, `userId`
- `accessToken`, `refreshToken`, `idToken`
- `accessTokenExpiresAt`, `refreshTokenExpiresAt`
- `scope`, `password`, `createdAt`, `updatedAt`

### Account Sync Job Table (camelCase columns)
- `id`, `accountId` (unique, FK to account)
- `status` (VARCHAR: pending/processing/completed/failed)
- `attempts`, `lastError`
- `createdAt`, `updatedAt`, `processedAt`

### Email Sync Job Table (camelCase columns)
- `id`, `accountId` (FK to account)
- `status` (VARCHAR: pending/processing/synced/failed)
- `syncType` (VARCHAR: initial/incremental/webhook)
- `emailsFetched`, `pageToken`, `lastSyncedAt`
- `attempts`, `lastError`
- `createdAt`, `updatedAt`, `processedAt`

**Note**: Status is stored as VARCHAR (not enum) for easier schema evolution, with CHECK constraint for validation.

## Available Commands

```bash
# Development
make build              # Build the application
make run                # Run the application
make clean              # Clean build artifacts

# Dependencies
make deps               # Download Go dependencies

# Migrations
make migrate-install    # Install golang-migrate CLI
make migrate-up         # Apply all pending migrations
make migrate-down       # Rollback last migration
make migrate-status     # Show current migration version
make migrate-create name=your_migration  # Create new migration

# Testing
make test               # Run all tests
make test-coverage      # Run tests with coverage report
```

## Testing

### Unit Tests

Run all tests:
```bash
make test
```

Generate coverage report:
```bash
make test-coverage
# Opens coverage.html in browser
```

**Current Coverage:**
- Config: 100%
- Repository: 85%
- Service: 100%

### Integration Testing

Insert test account:
```bash
psql "$DATABASE_URL" -c "
INSERT INTO account (
    id, \"accountId\", \"providerId\", \"userId\", 
    \"accessToken\", \"refreshToken\", \"accessTokenExpiresAt\",
    \"createdAt\", \"updatedAt\"
)
VALUES (
    'test-' || gen_random_uuid()::text,
    'acc-google-123',
    'google',
    'user-123',
    'ya29.test_token',
    'refresh_token',
    NOW() + INTERVAL '1 hour',
    NOW(),
    NOW()
);
"
```

Check job status:
```bash
psql "$DATABASE_URL" -c "SELECT * FROM account_sync_job ORDER BY \"createdAt\" DESC LIMIT 5;"
```

View watcher logs:
```
Found 1 pending job(s)
Processing job <id> for account <account_id>
Processing account: <account_id> (user: <user_id>)
Successfully completed job <id>
```

## Architecture Decisions

- **Polling vs LISTEN/NOTIFY**: Chose polling for MVP simplicity and reliability
- **Trigger-based job creation**: Ensures no missed accounts even during downtime
- **VARCHAR status over ENUM**: Easier schema evolution without ALTER TYPE migrations
- **camelCase columns**: Consistent with Prisma/frontend schema
- **Graceful shutdown**: Completes current job before exit
- **Retry logic**: Failed jobs retry up to 3 times before marking as failed

## Email Sync Strategy

### Fair Round-Robin with New Account Priority
- **New accounts** (`lastSyncedAt = NULL`): Get picked first
- **After first batch**: Join the fair pool with everyone else
- **Fair pool**: Oldest `lastSyncedAt` gets picked next
- **Query**: `ORDER BY "lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC`

### Fetching Behavior
- Fetches **50 emails per batch** to minimize memory usage
- Fetches in **reverse chronological order** (newest first) for recent payment dues
- Processes **one account at a time** (round-robin)
- **Resumes from last page** if interrupted
- Filters: **inbox only**, **excludes spam and social category**, **primary recipient only (no CC)**
- Limits: **10,000 emails max** or **1 year of history** per account

### Job Lifecycle
```
pending → processing → processing → ... → synced
   ↓           ↓
failed ←  failed
   ↓
processing (retry)
```

**State Transitions:**
- **pending → processing**: Watcher picks job
- **processing → processing**: Partial success (more pages to fetch)
- **processing → synced**: Complete success (all emails fetched)
- **processing → failed**: Error during processing
- **failed → processing**: Watcher picks failed job for retry

**Key Points:**
- Partial success stays in `processing` (not pending)
- All failures go to `failed` status
- `lastSyncedAt` updated on both success AND failure (prevents queue blocking)
- Watcher fetches `pending`, `failed`, AND `processing` jobs
- Jobs stuck in `processing` (from crashes) are automatically retried
- Infinite retry: failed jobs are picked up again in next cycle
- Round-robin fairness: oldest `lastSyncedAt` (or NULL) gets picked first

### Round-Robin Fairness
- After processing, `lastSyncedAt` is updated to NOW()
- Job goes to **back of queue** (oldest `lastSyncedAt` goes first)
- Prevents immediate re-processing of same account
- Ensures all accounts get equal turns

### Token Management
- Checks token expiry before each API call
- Auto-refreshes if expired or within 5 minutes of expiry
- Updates account table with new tokens and expiry times

## Gmail API Integration

### Setup

1. **Get OAuth2 credentials** from [Google Cloud Console](https://console.cloud.google.com/):
   - Create a project
   - Enable Gmail API
   - Create OAuth 2.0 Client ID
   - Add to `.env`:
     ```
     GOOGLE_CLIENT_ID="your-client-id.apps.googleusercontent.com"
     GOOGLE_CLIENT_SECRET="your-client-secret"
     ```

2. **Features implemented**:
   - ✅ OAuth2 token refresh with automatic expiry checking
   - ✅ Gmail messages.list API (with pagination)
   - ✅ Gmail messages.get API (full message details)
   - ✅ Email body extraction (text/plain and text/html)
   - ✅ Email header extraction (from, to, cc, bcc, subject, date)
   - ✅ Attachment metadata extraction
   - ✅ Raw headers and payload parsing (JSONB)
   - ✅ Email date parsing (multiple formats)
   - ✅ Token storage and updates in database

## LLM Payment Extraction

### How It Works

1. **Email Sync**: Fetches message IDs from Gmail (lightweight, fast)
2. **LLM Sync**: Fetches full emails on-demand and sends to OpenRouter
3. **Payment Extraction**: LLM extracts structured payment data with 85% confidence threshold
4. **Storage**: Valid payments stored in payments table

### Extracted Payment Fields

**Required fields** (if any cannot be inferred with ≥85% confidence, returns null):
| Field | Description |
|-------|-------------|
| `merchant` | Business/entity name exactly as it appears in email |
| `amount` | Total due amount (numeric, positive even for refunds) |
| `currency` | ISO 4217 code (INR, USD, EUR, etc.) - must be explicitly inferable |
| `date` | ISO 8601 with timezone, contextual to status |
| `status` | draft, scheduled, upcoming, due, overdue, processing, partially_paid, paid, failed, refunded, cancelled, written_off |

**Optional fields** (null if not inferable):
| Field | Description |
|-------|-------------|
| `description` | What the payment is for |
| `recurrence` | daily, weekly, biweekly, monthly, bimonthly, quarterly, semiannual, annual |
| `category` | subscription, utility, emi, credit_card_bill, loan, insurance, rent, misc |
| `metadata` | Flat JSON with additional details (invoice_number, card_last_four, etc.) |

**Status Logic:**
- `upcoming`: Due date >24hrs away
- `due`: Due date within 24hrs
- `overdue`: Due date has passed
- Other statuses inferred from email context (paid, failed, refunded, etc.)

**Note:** `credit_card_bill` category is ONLY for credit card dues/statements, not payments made via credit card.

### Data Sent to LLM

For each email, we send:
- **current_time**: Current timestamp with timezone (for status inference)
- **from**: Sender email address
- **subject**: Email subject line
- **body**: Plain text body (first 5,000 characters)

**Gmail Query Filters:**
- `in:inbox` - Only inbox emails
- `-in:spam` - Exclude spam
- `-category:social` - Exclude social media notifications (Facebook, LinkedIn, Twitter, etc.)
- `deliveredto:me` - Only emails where user is primary recipient (excludes CC'ed emails)
- `{keywords}` - Comprehensive payment keyword filter (see below)
- `after:YYYY/MM/DD` - Time-based filter (1 year for initial sync)

**Payment Keywords (comprehensive list):**
- Core: invoice, bill, payment, paid, pay, due, overdue, outstanding, balance, amount, total, charge, charged
- Subscription: subscription, renewal, renew, recurring, membership, plan, premium, upgrade, downgrade
- Documents: receipt, statement, confirmation, order, purchase, transaction, refund
- Reminders: reminder, notice, alert, expiring, expires, expiry, deadline
- Regional: emi, installment, instalment, booking, reservation
- Actions: renewing, billing, billed, autopay, auto-pay

**What gets excluded:**
- Social media notifications
- Spam emails
- Emails where user is only CC'ed
- Sent emails
- Drafts and trash
- Emails without payment-related keywords

**What gets included:**
- Primary inbox emails with payment keywords
- Promotions (subscription renewals, e-commerce invoices)
- Updates (order confirmations, booking confirmations)
- All emails where user is in "To" field with payment terms

**Cost Savings:**
- Keyword filter reduces emails by 70-80%
- Estimated: 1000 emails → 200-300 LLM calls
- Maintains high coverage (~85-90%) of actual payment emails

**Why plain text?**
- Smaller token count (no HTML markup)
- Faster LLM processing
- Lower costs
- Cleaner data without HTML tags and styling

**Why 5,000 character limit?**
- DDoS protection against extremely long emails
- Efficient token usage (~1,250 tokens per email)
- Payment info is typically in first 2,000 characters
- Allows 3 emails per batch within reasonable limits

### OpenRouter Configuration

Add to `.env`:
```
OPENROUTER_API_KEY="sk-or-v1-..."
```

The service uses your OpenRouter account's default model. You can change the model in OpenRouter settings without code changes.

## Next Steps

1. **Setup Gmail webhook** for real-time email notifications

2. **Implement payment deduplication** across multiple tables

3. **Add payment status tracking** and notifications

## Technologies

- **Go 1.21+**: Backend service
- **PostgreSQL**: Database with triggers
- **GORM**: ORM for type-safe, SQL-injection-proof database queries
- **golang-migrate**: Database migrations (CLI and programmatic)
- **Prisma**: Schema management (frontend)

## Conventions

- [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
- [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

## Collaborators

- **Vishnuprakash P**
  - [GitHub](https://github.com/vishkashpvp)
  - [Mail](mailto:vishkash.k@gmail.com)

- **Hassain Saheb S**
  - [GitHub](https://github.com/hafeezzshs)
  - [Mail](mailto:hafeezz.dev@gmail.com)
