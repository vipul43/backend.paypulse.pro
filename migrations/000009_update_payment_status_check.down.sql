-- Revert payment status CHECK constraint
-- Add back time-dependent statuses (upcoming, due, overdue) and remove 'unpaid'

-- First, convert 'unpaid' to 'due' (default fallback)
-- Must be done BEFORE updating the constraint
UPDATE payment SET status = 'due' WHERE status = 'unpaid';

-- Drop new constraint
ALTER TABLE payment DROP CONSTRAINT IF EXISTS payment_status_check;

-- Add back original constraint
ALTER TABLE payment ADD CONSTRAINT payment_status_check CHECK (
    status IN ('draft', 'scheduled', 'upcoming', 'due', 'overdue', 'processing', 
               'partially_paid', 'paid', 'failed', 'refunded', 'cancelled', 'written_off')
);
