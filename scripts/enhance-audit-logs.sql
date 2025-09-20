-- Enhanced audit_logs table for comprehensive LLM request tracking
-- This shows what a production-ready gateway should track

-- Add columns to existing audit_logs table
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS model VARCHAR(100);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS completion_tokens INTEGER;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS total_tokens INTEGER;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS cost_usd DECIMAL(10, 6);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS temperature DECIMAL(3, 2);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS max_tokens INTEGER;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS error_message TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS api_key_id VARCHAR(100);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_id VARCHAR(100);

-- Create a separate table for request/response content
CREATE TABLE IF NOT EXISTS request_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    audit_log_id UUID REFERENCES audit_logs(id) ON DELETE CASCADE,
    request_body JSONB NOT NULL,  -- Full request including messages array
    response_body JSONB,          -- Full response from provider
    request_headers JSONB,         -- Headers sent to provider
    response_headers JSONB,        -- Headers received from provider
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create a table for message history (denormalized for analytics)
CREATE TABLE IF NOT EXISTS message_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    audit_log_id UUID REFERENCES audit_logs(id) ON DELETE CASCADE,
    message_index INTEGER NOT NULL,
    role VARCHAR(50) NOT NULL,     -- 'system', 'user', 'assistant', 'function'
    content TEXT,                   -- The actual message content
    function_name VARCHAR(255),     -- If it's a function call
    function_args JSONB,            -- Function arguments if applicable
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create a table for tracking costs
CREATE TABLE IF NOT EXISTS cost_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    total_requests INTEGER DEFAULT 0,
    total_prompt_tokens BIGINT DEFAULT 0,
    total_completion_tokens BIGINT DEFAULT 0,
    total_cost_usd DECIMAL(10, 6) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, model, date)
);

-- Create a table for PII detection results
CREATE TABLE IF NOT EXISTS pii_detections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    audit_log_id UUID REFERENCES audit_logs(id) ON DELETE CASCADE,
    detection_type VARCHAR(100),   -- 'email', 'phone', 'ssn', 'credit_card', etc.
    redacted_text TEXT,            -- The redacted version
    original_hash VARCHAR(64),     -- SHA256 of original for verification
    location VARCHAR(50),          -- 'request' or 'response'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_request_logs_audit_id ON request_logs(audit_log_id);
CREATE INDEX IF NOT EXISTS idx_message_logs_audit_id ON message_logs(audit_log_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_model ON audit_logs(model);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_cost_tracking_date ON cost_tracking(date DESC);
CREATE INDEX IF NOT EXISTS idx_pii_detections_audit_id ON pii_detections(audit_log_id);

-- Create a view for easy querying
CREATE OR REPLACE VIEW gateway_metrics AS
SELECT 
    al.provider,
    al.model,
    COUNT(*) as request_count,
    AVG(al.processing_time_ms) as avg_latency_ms,
    SUM(al.total_tokens) as total_tokens_used,
    SUM(al.cost_usd) as total_cost,
    COUNT(CASE WHEN al.status_code >= 400 THEN 1 END) as error_count,
    DATE_TRUNC('hour', al.created_at) as hour
FROM audit_logs al
WHERE al.created_at > NOW() - INTERVAL '24 hours'
GROUP BY al.provider, al.model, DATE_TRUNC('hour', al.created_at)
ORDER BY hour DESC;

-- Example of what would be stored:
/*
-- In audit_logs:
INSERT INTO audit_logs (
    request_id, provider, model, method, path,
    status_code, processing_time_ms,
    prompt_tokens, completion_tokens, total_tokens,
    cost_usd, temperature, max_tokens
) VALUES (
    'req_123', 'openai', 'gpt-4o-mini', 'POST', '/v1/openai/chat/completions',
    200, 1234,
    150, 50, 200,
    0.000060, 0.7, 1000
);

-- In request_logs:
INSERT INTO request_logs (audit_log_id, request_body, response_body) VALUES (
    'audit_id_here',
    '{"model": "gpt-4o-mini", "messages": [{"role": "user", "content": "Hello"}], "temperature": 0.7}',
    '{"choices": [{"message": {"content": "Hi there!"}}], "usage": {"prompt_tokens": 150, "completion_tokens": 50}}'
);

-- In message_logs (for analytics):
INSERT INTO message_logs (audit_log_id, message_index, role, content) VALUES
    ('audit_id_here', 0, 'user', 'Hello'),
    ('audit_id_here', 1, 'assistant', 'Hi there! How can I help you today?');
*/

