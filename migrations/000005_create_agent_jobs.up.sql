CREATE TABLE agent_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    job_type VARCHAR(50) NOT NULL,     -- reconciliation, anomaly_scan, spend_report, query
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    input JSONB NOT NULL,
    output JSONB,
    tokens_used INT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
