-- blayzen-sip Database Schema
-- Version: 001_initial

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- Accounts Table
-- =============================================================================
-- Accounts for multi-tenancy support
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(64) UNIQUE NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for API key lookups
CREATE INDEX IF NOT EXISTS idx_accounts_api_key ON accounts(api_key) WHERE active = true;

-- =============================================================================
-- SIP Routes Table
-- =============================================================================
-- Inbound routing rules that map SIP calls to Blayzen agents
CREATE TABLE IF NOT EXISTS sip_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    priority INT DEFAULT 0,
    
    -- Match conditions (NULL means match all)
    match_to_user VARCHAR(255),           -- DID/extension to match
    match_from_user VARCHAR(255),         -- Caller ID to match
    match_sip_header VARCHAR(255),        -- Custom SIP header name (e.g., X-Customer-Tier)
    match_sip_header_value VARCHAR(255),  -- Custom SIP header value pattern
    
    -- Action: where to route the call
    websocket_url VARCHAR(512) NOT NULL,  -- Blayzen agent WebSocket URL
    
    -- Metadata to pass to the agent
    custom_data JSONB DEFAULT '{}',
    
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for route lookups
CREATE INDEX IF NOT EXISTS idx_routes_account ON sip_routes(account_id) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_routes_to_user ON sip_routes(match_to_user) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_routes_priority ON sip_routes(account_id, priority DESC) WHERE active = true;

-- =============================================================================
-- SIP Trunks Table
-- =============================================================================
-- Outbound SIP trunk configurations
CREATE TABLE IF NOT EXISTS sip_trunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    
    -- SIP trunk connection settings
    host VARCHAR(255) NOT NULL,           -- SIP server hostname/IP
    port INT DEFAULT 5060,                -- SIP port
    transport VARCHAR(10) DEFAULT 'udp',  -- udp, tcp, tls
    
    -- Authentication (optional)
    username VARCHAR(255),
    password VARCHAR(255),                -- Encrypted in production
    
    -- Caller ID settings
    from_user VARCHAR(255),               -- From user part
    from_host VARCHAR(255),               -- From host part (defaults to trunk host)
    
    -- Advanced settings
    register BOOLEAN DEFAULT false,       -- Whether to register with the trunk
    register_interval INT DEFAULT 3600,   -- Registration interval in seconds
    
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for trunk lookups
CREATE INDEX IF NOT EXISTS idx_trunks_account ON sip_trunks(account_id) WHERE active = true;

-- =============================================================================
-- Call Logs Table
-- =============================================================================
-- CDR (Call Detail Records)
CREATE TABLE IF NOT EXISTS call_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    
    -- SIP identifiers
    call_id VARCHAR(255) NOT NULL,        -- SIP Call-ID header
    
    -- Call direction
    direction VARCHAR(10) NOT NULL,       -- 'inbound' or 'outbound'
    
    -- Call parties
    from_uri VARCHAR(512),                -- Full From URI
    to_uri VARCHAR(512),                  -- Full To URI
    from_user VARCHAR(255),               -- From user part
    to_user VARCHAR(255),                 -- To user part
    
    -- Routing information
    route_id UUID REFERENCES sip_routes(id) ON DELETE SET NULL,
    trunk_id UUID REFERENCES sip_trunks(id) ON DELETE SET NULL,
    websocket_url VARCHAR(512),           -- Agent URL used
    
    -- Call status
    status VARCHAR(50) NOT NULL DEFAULT 'initiated',
    -- Status values: initiated, ringing, answered, completed, failed, cancelled
    
    -- Timing
    initiated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ringing_at TIMESTAMP WITH TIME ZONE,
    answered_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INT,                 -- Total duration from answer to end
    
    -- Termination
    hangup_cause VARCHAR(50),             -- SIP response code or reason
    hangup_party VARCHAR(20),             -- 'caller', 'callee', 'system'
    
    -- Metadata
    custom_data JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for call log queries
CREATE INDEX IF NOT EXISTS idx_calls_account ON call_logs(account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_calls_call_id ON call_logs(call_id);
CREATE INDEX IF NOT EXISTS idx_calls_status ON call_logs(status) WHERE status IN ('initiated', 'ringing', 'answered');
CREATE INDEX IF NOT EXISTS idx_calls_created ON call_logs(created_at DESC);

-- =============================================================================
-- Updated At Trigger
-- =============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to tables with updated_at
DROP TRIGGER IF EXISTS update_accounts_updated_at ON accounts;
CREATE TRIGGER update_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_routes_updated_at ON sip_routes;
CREATE TRIGGER update_routes_updated_at
    BEFORE UPDATE ON sip_routes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_trunks_updated_at ON sip_trunks;
CREATE TRIGGER update_trunks_updated_at
    BEFORE UPDATE ON sip_trunks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

