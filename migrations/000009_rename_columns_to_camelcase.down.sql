-- Revert columns from camelCase back to snake_case

-- ============================================================================
-- account_sync_job table
-- ============================================================================
ALTER TABLE account_sync_job RENAME COLUMN "accountId" TO account_id;
ALTER TABLE account_sync_job RENAME COLUMN "lastError" TO last_error;
ALTER TABLE account_sync_job RENAME COLUMN "createdAt" TO created_at;
ALTER TABLE account_sync_job RENAME COLUMN "updatedAt" TO updated_at;
ALTER TABLE account_sync_job RENAME COLUMN "processedAt" TO processed_at;

-- Drop and recreate indexes with original column names
DROP INDEX IF EXISTS idx_account_sync_job_status_created;
CREATE INDEX idx_account_sync_job_status_created ON account_sync_job(status, created_at);

-- ============================================================================
-- email_sync_job table
-- ============================================================================
ALTER TABLE email_sync_job RENAME COLUMN "accountId" TO account_id;
ALTER TABLE email_sync_job RENAME COLUMN "syncType" TO sync_type;
ALTER TABLE email_sync_job RENAME COLUMN "emailsFetched" TO emails_fetched;
ALTER TABLE email_sync_job RENAME COLUMN "pageToken" TO page_token;
ALTER TABLE email_sync_job RENAME COLUMN "lastSyncedAt" TO last_synced_at;
ALTER TABLE email_sync_job RENAME COLUMN "lastError" TO last_error;
ALTER TABLE email_sync_job RENAME COLUMN "createdAt" TO created_at;
ALTER TABLE email_sync_job RENAME COLUMN "updatedAt" TO updated_at;
ALTER TABLE email_sync_job RENAME COLUMN "processedAt" TO processed_at;

-- Drop and recreate indexes with original column names
DROP INDEX IF EXISTS idx_email_sync_job_round_robin;
DROP INDEX IF EXISTS idx_email_sync_job_account;
CREATE INDEX idx_email_sync_job_round_robin ON email_sync_job(status, last_synced_at ASC NULLS FIRST, created_at ASC);
CREATE INDEX idx_email_sync_job_account ON email_sync_job(account_id);

-- ============================================================================
-- llm_sync_job table
-- ============================================================================
ALTER TABLE llm_sync_job RENAME COLUMN "accountId" TO account_id;
ALTER TABLE llm_sync_job RENAME COLUMN "messageId" TO message_id;
ALTER TABLE llm_sync_job RENAME COLUMN "lastSyncedAt" TO last_synced_at;
ALTER TABLE llm_sync_job RENAME COLUMN "lastError" TO last_error;
ALTER TABLE llm_sync_job RENAME COLUMN "createdAt" TO created_at;
ALTER TABLE llm_sync_job RENAME COLUMN "updatedAt" TO updated_at;
ALTER TABLE llm_sync_job RENAME COLUMN "processedAt" TO processed_at;

-- Drop and recreate indexes with original column names
DROP INDEX IF EXISTS idx_llm_sync_job_status_last_synced;
DROP INDEX IF EXISTS idx_llm_sync_job_account_id;
DROP INDEX IF EXISTS idx_llm_sync_job_message_id;
DROP INDEX IF EXISTS idx_llm_sync_job_created_at;
CREATE INDEX idx_llm_sync_job_status_last_synced ON llm_sync_job(status, last_synced_at ASC NULLS FIRST);
CREATE INDEX idx_llm_sync_job_account_id ON llm_sync_job(account_id);
CREATE UNIQUE INDEX idx_llm_sync_job_message_id ON llm_sync_job(message_id);
CREATE INDEX idx_llm_sync_job_created_at ON llm_sync_job(created_at DESC);

-- ============================================================================
-- payment table
-- ============================================================================
ALTER TABLE payment RENAME COLUMN "accountId" TO account_id;
ALTER TABLE payment RENAME COLUMN "externalReference" TO external_reference;
ALTER TABLE payment RENAME COLUMN "rawLlmResponse" TO raw_llm_response;
ALTER TABLE payment RENAME COLUMN "createdAt" TO created_at;
ALTER TABLE payment RENAME COLUMN "updatedAt" TO updated_at;

-- Drop and recreate indexes with original column names
DROP INDEX IF EXISTS idx_payment_account_id;
DROP INDEX IF EXISTS idx_payment_date;
DROP INDEX IF EXISTS idx_payment_account_date;
CREATE INDEX idx_payment_account_id ON payment(account_id);
CREATE INDEX idx_payment_date ON payment(date DESC);
CREATE INDEX idx_payment_account_date ON payment(account_id, date DESC);
