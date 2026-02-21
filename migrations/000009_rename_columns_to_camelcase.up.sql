-- Rename columns from snake_case to camelCase for consistency with frontend/Prisma schema
-- Only affects tables created by this application: account_sync_job, email_sync_job, llm_sync_job, payment

-- ============================================================================
-- account_sync_job table
-- ============================================================================
ALTER TABLE account_sync_job RENAME COLUMN account_id TO "accountId";
ALTER TABLE account_sync_job RENAME COLUMN last_error TO "lastError";
ALTER TABLE account_sync_job RENAME COLUMN created_at TO "createdAt";
ALTER TABLE account_sync_job RENAME COLUMN updated_at TO "updatedAt";
ALTER TABLE account_sync_job RENAME COLUMN processed_at TO "processedAt";

-- Drop and recreate indexes with new column names
DROP INDEX IF EXISTS idx_account_sync_job_status_created;
CREATE INDEX idx_account_sync_job_status_created ON account_sync_job(status, "createdAt");

-- ============================================================================
-- email_sync_job table
-- ============================================================================
ALTER TABLE email_sync_job RENAME COLUMN account_id TO "accountId";
ALTER TABLE email_sync_job RENAME COLUMN sync_type TO "syncType";
ALTER TABLE email_sync_job RENAME COLUMN emails_fetched TO "emailsFetched";
ALTER TABLE email_sync_job RENAME COLUMN page_token TO "pageToken";
ALTER TABLE email_sync_job RENAME COLUMN last_synced_at TO "lastSyncedAt";
ALTER TABLE email_sync_job RENAME COLUMN last_error TO "lastError";
ALTER TABLE email_sync_job RENAME COLUMN created_at TO "createdAt";
ALTER TABLE email_sync_job RENAME COLUMN updated_at TO "updatedAt";
ALTER TABLE email_sync_job RENAME COLUMN processed_at TO "processedAt";

-- Drop and recreate indexes with new column names
DROP INDEX IF EXISTS idx_email_sync_job_round_robin;
DROP INDEX IF EXISTS idx_email_sync_job_account;
CREATE INDEX idx_email_sync_job_round_robin ON email_sync_job(status, "lastSyncedAt" ASC NULLS FIRST, "createdAt" ASC);
CREATE INDEX idx_email_sync_job_account ON email_sync_job("accountId");

-- ============================================================================
-- llm_sync_job table
-- ============================================================================
ALTER TABLE llm_sync_job RENAME COLUMN account_id TO "accountId";
ALTER TABLE llm_sync_job RENAME COLUMN message_id TO "messageId";
ALTER TABLE llm_sync_job RENAME COLUMN last_synced_at TO "lastSyncedAt";
ALTER TABLE llm_sync_job RENAME COLUMN last_error TO "lastError";
ALTER TABLE llm_sync_job RENAME COLUMN created_at TO "createdAt";
ALTER TABLE llm_sync_job RENAME COLUMN updated_at TO "updatedAt";
ALTER TABLE llm_sync_job RENAME COLUMN processed_at TO "processedAt";

-- Drop and recreate indexes with new column names
DROP INDEX IF EXISTS idx_llm_sync_job_status_last_synced;
DROP INDEX IF EXISTS idx_llm_sync_job_account_id;
DROP INDEX IF EXISTS idx_llm_sync_job_message_id;
DROP INDEX IF EXISTS idx_llm_sync_job_created_at;
CREATE INDEX idx_llm_sync_job_status_last_synced ON llm_sync_job(status, "lastSyncedAt" ASC NULLS FIRST);
CREATE INDEX idx_llm_sync_job_account_id ON llm_sync_job("accountId");
CREATE UNIQUE INDEX idx_llm_sync_job_message_id ON llm_sync_job("messageId");
CREATE INDEX idx_llm_sync_job_created_at ON llm_sync_job("createdAt" DESC);

-- ============================================================================
-- payment table
-- ============================================================================
ALTER TABLE payment RENAME COLUMN account_id TO "accountId";
ALTER TABLE payment RENAME COLUMN external_reference TO "externalReference";
ALTER TABLE payment RENAME COLUMN raw_llm_response TO "rawLlmResponse";
ALTER TABLE payment RENAME COLUMN created_at TO "createdAt";
ALTER TABLE payment RENAME COLUMN updated_at TO "updatedAt";

-- Drop and recreate indexes with new column names
DROP INDEX IF EXISTS idx_payment_account_id;
DROP INDEX IF EXISTS idx_payment_date;
DROP INDEX IF EXISTS idx_payment_account_date;
CREATE INDEX idx_payment_account_id ON payment("accountId");
CREATE INDEX idx_payment_date ON payment(date DESC);
CREATE INDEX idx_payment_account_date ON payment("accountId", date DESC);
