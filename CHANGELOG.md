# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Account watcher service with polling mechanism (10-second interval)
- PostgreSQL trigger to automatically create sync jobs on account insert
- Account sync job table with status tracking (pending/processing/completed/failed)
- Email sync job table with fair round-robin processing
- Email table for storing raw email data (for LLM fine-tuning)
- Fair round-robin: new accounts (last_synced_at=NULL) get picked first, then oldest synced jobs
- No priority field: uses last_synced_at for natural round-robin ordering
- Round-robin email fetching: processes one account at a time
- Job lifecycle: pending→processing→synced (email) or completed (account)
- Sync types: initial (historical), incremental (manual re-sync), webhook (real-time)
- Status types: pending, processing, synced, completed, failed
- Synced status: marks completion of historical sync, ready for webhook
- Completed status: account setup complete, email sync job created
- Fair round-robin: last_synced_at updated after each batch to push job to back of queue
- Reverse chronological fetching: newest emails first for recent payment dues
- Pagination support: fetches 50 emails per batch, resumes from last page token
- Email sync limits: max 10,000 emails or 1 year of history per account
- Token refresh logic with automatic expiry checking (5-minute buffer)
- UUID-based IDs: all job IDs use UUIDs for flexibility
- Gmail API integration: OAuth2 token refresh, email fetching, comprehensive data extraction
- Token management: automatic refresh and database updates
- Email parsing: extracts from, to, cc, bcc, subject, body (text/html), attachments, headers
- Email storage: stores complete email data including raw headers, payload, and attachments metadata
- Pagination: Gmail API pagination with nextPageToken support
- GetProcessingJobs: fetches stuck jobs from crashes/restarts
- Infinite retry with failed status: jobs retry forever, failures go to failed status
- Graceful shutdown handling with context cancellation
- Database migrations using golang-migrate
- Makefile commands for build, run, migration management, testing, formatting, and linting
- CI/CD pipeline with GitHub Actions (format check, lint, test, build)
- CASCADE delete on account removal (automatically removes sync jobs)
- Clean architecture with separation of concerns (config, database, models, repository, service, watcher)
- Type-safe enums in Go code with VARCHAR storage in database
- Connection pooling configuration
- Environment-based configuration via .env file
- Comprehensive unit tests for all layers (config, models, repository, service)
- Test coverage reporting with HTML output
- Mock-based testing using go-sqlmock for database operations
- LLM sync job table for tracking payment extraction from emails
- Payment table for storing extracted payment information
- Three-stage job pipeline: Account Sync → Email Sync → LLM Sync
- OpenRouter client for LLM-based payment extraction
- Batch processing: processes 100 LLM jobs at a time
- Email sync now fetches only message IDs (lightweight, fast)
- LLM sync fetches full emails on demand and sends to LLM
- Payment extraction with validation of required fields
- Raw LLM response storage in payments table for audit
- Non-payment emails marked as completed without creating payment
- Round-robin batch processing across all accounts
- LLMSyncJobRepository with GetPendingJobs, GetFailedJobs, GetProcessingJobs
- PaymentRepository with Create and BulkCreate methods
- LLMProcessor service for batch email processing and payment extraction
- Gmail client FetchMessageIDs method for lightweight message ID fetching
- OpenRouter API integration with batch support
- OPENROUTER_API_KEY configuration variable
- Unit tests for LLMSyncJob model (status constants and structure)
- Unit tests for Payment model (status constants, recurrence constants, and structure)
- Unit tests for OpenRouter client (cleanJSONResponse and isValidPayment functions)
- Test coverage for JSON cleaning with various formats (markdown, plain text, whitespace)
- Test coverage for payment validation with edge cases (missing fields, invalid amounts)
- Account UPDATE trigger: now triggers account sync job reset when account is updated
- Account sync job reset on update: sets status to pending, clears last_error and processed_at
- Full re-sync on account update: entire sync process (account → email → LLM) reruns from beginning
- CI migrations test job: spins up PostgreSQL 18, runs all migrations up/down/up to verify integrity

### Changed

- Database column naming: uses camelCase to match Prisma/frontend schema
- Status field from ENUM type to VARCHAR(50) with CHECK constraint for easier schema evolution
- AccountProcessor now uses interface for better testability
- Watcher now handles both account sync and email sync jobs
- Account setup creates email sync job after completion
- All SQL queries use quoted camelCase column names (e.g., "accountId", "userId")
- Removed max retry limit: jobs now retry infinitely
- Error handlers: failures go to failed status (not pending) for clear state tracking
- Watcher fetches pending, failed, AND processing jobs for crash recovery
- Email sync partial success: stays in processing status (not pending)
- ProcessEmailSyncJob: updates job object in-place to avoid extra DB queries
- Failed jobs: update last_synced_at on failure to prevent queue blocking
- Round-robin: queries pre-sorted by last_synced_at, no additional sorting needed
- Improved error messages: specific validation for missing access/refresh tokens
- Email sync job now creates LLM sync jobs instead of processing emails directly
- EmailProcessor modified to fetch message IDs only (not full emails)
- Gmail client interface updated with FetchMessageIDs method
- Watcher now processes three job types: account, email, and LLM sync
- Main application wired with LLM processor and OpenRouter client
- Database schema uses snake_case for all column names (consistent with existing tables)
- llm_sync_job table: account_id, message_id, last_synced_at, last_error, created_at, updated_at, processed_at
- payment table: account_id, external_reference, raw_llm_response, created_at, updated_at
- All repository queries updated to use snake_case column names
- OpenRouter client now uses account default model instead of hardcoded model
- Model parameter is optional - if not set, uses the default model configured in OpenRouter account settings
- Added SetModel() method to optionally override the default model if needed
- Increased OpenRouter API timeout from 2 minutes to 5 minutes (free models are slow)
- Reduced LLM batch size from 100 to 10 jobs per cycle (prevents timeout with slow free models)
- Each batch now processes ~10 emails taking 3-10 minutes total (manageable within timeout)
- Further reduced LLM batch size from 10 to 3 jobs per cycle to prevent timeouts
- Each batch now processes ~3 emails taking 1.5-3 minutes (well within 5-minute timeout)
- Watcher processes batches every 10 seconds, so still makes steady progress
- Renamed environment variables for frontend consistency: GMAIL_CLIENT_ID → GOOGLE_CLIENT_ID, GMAIL_CLIENT_SECRET → GOOGLE_CLIENT_SECRET
- Updated config struct fields: GmailClientID → GoogleClientID, GmailClientSecret → GoogleClientSecret
- Updated all references in main.go, .env, .env.example, README.md, and tests
- Renamed trigger function from `create_account_sync_job()` to `handle_account_sync_job()` to reflect dual purpose
- Renamed trigger from `account_insert_trigger` to `account_sync_trigger` for clarity
- Trigger now fires on both INSERT and UPDATE operations on account table
- Improved LLM payment extraction prompt with 85% confidence threshold
- Payment schema updated: `merchant_name` → `merchant`, `due` → `date`
- Removed `external_reference` from LLM response (now captured in metadata)
- Date field now includes timezone (ISO 8601 with offset)
- Currency no longer defaults to INR - must be explicitly inferred from email
- Status values expanded: added `draft`, `scheduled`, `due`, `processing`, `partially_paid`, `failed`, `refunded`, `written_off`
- Recurrence values updated: `yearly` → `annual`, added `biweekly`, `bimonthly`
- LLM prompt now receives structured input: `current_time`, `from`, `subject`, `body`
- Status inference uses current timestamp comparison: `upcoming` (>24hrs), `due` (within 24hrs), `overdue` (past)
- Promotional/marketing emails now return `null` instead of invalid payment data
- Optional fields (`description`, `category`) are now pointers in PaymentData struct
- All additional inferred details (invoice_number, subscription_id, card_last_four, etc.) go to flat metadata JSON
- LLM payment extraction now uses plain text email body instead of HTML
- Reduced token count and improved LLM processing efficiency by removing HTML markup
- Cleaner data sent to LLM without HTML tags, styling, and formatting noise
- Email body truncated to 5,000 characters for DDoS protection and efficient token usage
- Payment information (typically in first 2,000 chars) is preserved while preventing abuse
- Gmail query now excludes social category emails (Facebook, LinkedIn, Twitter notifications)
- Added comprehensive keyword filter for payment-related emails (70-80% cost reduction)
- Keywords: invoice, bill, payment, due, subscription, receipt, order, purchase, membership, emi, renewal, charge, and 30+ more
- Reduces LLM processing costs while maintaining high coverage of payment emails
- Query filter: `in:inbox -in:spam -category:social deliveredto:me {keywords...}`
- Removed default value 'INR' from payment.currency column
- Currency must now be explicitly provided by LLM (no assumptions)
- Changed payment.date column from TIMESTAMP to TIMESTAMPTZ (timestamp with timezone)
- Migrated from raw SQL (database/sql + lib/pq) to GORM ORM for database operations
- All repositories now use GORM for type-safe, SQL-injection-proof queries
- Models updated with GORM tags for column mapping and table names
- Added JSONB custom type for PostgreSQL JSONB columns (metadata, raw_llm_response)
- Kept golang-migrate for manual migrations (CLI, Makefile) - GORM only used for queries
- Removed go-sqlmock tests (incompatible with GORM), kept interface-based service tests
- Account model moved from repository package to models package for consistency
- Renamed all columns in application tables from snake_case to camelCase for consistency with account table (Prisma/frontend schema)
- Tables affected: account_sync_job, email_sync_job, llm_sync_job, payment
- Column renames: account_id → accountId, created_at → createdAt, updated_at → updatedAt, etc.
- Recreated all indexes with new column names

### Removed

- Email table and all email storage functionality (simplified architecture for payment extraction focus)
- Environment-based configuration (no longer needed without email storage)
- Email table migration files (000004_create_email_table.up.sql and .down.sql)
- Email model (internal/models/email.go)
- Email repository (internal/repository/email_repository.go)
- Email storage logic from EmailProcessor
- EmailProcessor dependencies: emailRepo and environment parameters
- Configuration struct Environment field
- ENVIRONMENT variable from .env and .env.example

### Fixed

- Email job handler now passes job by pointer to preserve in-place updates from ProcessEmailSyncJob
- Re-ran migrations 000004 and 000005 to ensure llm_sync_job and payment tables are created
- Confirmed message_id has unique index to prevent duplicate message processing
- LLM processor now fetches emails directly by Gmail message ID instead of using incorrect query
- Added FetchEmailByID method to Gmail client for direct message retrieval
- Fixed "email not found" errors in LLM sync jobs
- Added JSON response cleaning to handle LLM responses wrapped in markdown code blocks
- Strips ```json and ``` markers before parsing JSON
- Prevents "invalid character '`'" parsing errors
- Improved JSON response cleaning to extract only the JSON object from LLM responses
- Now handles responses with explanatory text before/after the JSON
- Extracts content between first { and last } to get clean JSON
- Fixed all linting errors (errcheck and staticcheck)
- Added proper error handling for deferred tx.Rollback() calls
- Added blank identifier for intentionally ignored error returns
- Removed unused messages variable in BatchExtractPayments
- Gmail query now excludes CC'ed emails using `deliveredto:me` filter
- Only fetches emails where user is primary recipient (in "To" field)
- Prevents processing of emails where user is only CC'ed
- Email sync job `processed_at` now correctly updates when job completes (synced/completed/failed status)
- Gmail date parsing now handles timezone names in parentheses (e.g., "Fri, 12 Dec 2025 09:49:36 +0000 (UTC)")
- Date parser strips timezone name suffix before parsing to prevent "unable to parse date" errors
