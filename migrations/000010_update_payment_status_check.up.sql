-- Update payment status CHECK constraint
-- Remove time-dependent statuses (upcoming, due, overdue) and add 'unpaid'
-- Frontend will calculate display status (upcoming/due/overdue) based on current time vs date

-- Drop existing constraint FIRST (allows any status temporarily)
ALTER TABLE payment DROP CONSTRAINT IF EXISTS payment_status_check;

-- Migrate existing data: convert upcoming/due/overdue to unpaid
UPDATE payment SET status = 'unpaid' WHERE status IN ('upcoming', 'due', 'overdue');

-- Add new constraint with updated status values
ALTER TABLE payment ADD CONSTRAINT payment_status_check CHECK (
    status IN ('draft', 'scheduled', 'unpaid', 'processing', 
               'partially_paid', 'paid', 'failed', 'refunded', 'cancelled', 'written_off')
);
