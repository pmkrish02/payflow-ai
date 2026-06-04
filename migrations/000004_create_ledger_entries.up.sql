CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    account_id UUID NOT NULL REFERENCES accounts(id),
    entry_type VARCHAR(10) NOT NULL,   -- 'debit' or 'credit'
    amount BIGINT NOT NULL,            -- always positive, direction from entry_type
    balance_after BIGINT NOT NULL,     -- snapshot of account balance after this entry
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ledger_account ON ledger_entries(account_id, created_at);
CREATE INDEX idx_ledger_transaction ON ledger_entries(transaction_id);
