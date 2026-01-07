-- blayzen-sip Seed Data
-- Test data for development

-- =============================================================================
-- Default Account
-- =============================================================================
INSERT INTO accounts (id, name, api_key) VALUES 
    ('00000000-0000-0000-0000-000000000001', 'Default Account', 'test-api-key-12345')
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- Sample Inbound Routes
-- =============================================================================

-- Route calls to extension 1000 to echo bot
INSERT INTO sip_routes (account_id, name, match_to_user, websocket_url, priority) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Echo Bot', '1000', 'ws://host.docker.internal:8081/ws', 10)
ON CONFLICT DO NOTHING;

-- Route calls to extension 2000 to support agent
INSERT INTO sip_routes (account_id, name, match_to_user, websocket_url, priority, custom_data) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Support Agent', '2000', 'ws://host.docker.internal:8082/ws', 10, '{"department": "support"}')
ON CONFLICT DO NOTHING;

-- Route calls to extension 3000 to sales agent
INSERT INTO sip_routes (account_id, name, match_to_user, websocket_url, priority, custom_data) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Sales Agent', '3000', 'ws://host.docker.internal:8083/ws', 10, '{"department": "sales"}')
ON CONFLICT DO NOTHING;

-- VIP route based on custom SIP header
INSERT INTO sip_routes (account_id, name, match_sip_header, match_sip_header_value, websocket_url, priority, custom_data) VALUES
    ('00000000-0000-0000-0000-000000000001', 'VIP Route', 'X-Customer-Tier', 'vip', 'ws://host.docker.internal:8084/ws', 100, '{"tier": "vip", "priority_queue": true}')
ON CONFLICT DO NOTHING;

-- Catch-all route (lowest priority)
INSERT INTO sip_routes (account_id, name, websocket_url, priority) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Default Route', 'ws://host.docker.internal:8081/ws', 0)
ON CONFLICT DO NOTHING;

-- =============================================================================
-- Sample SIP Trunks
-- =============================================================================

-- Example SIP trunk (not functional, just for demonstration)
INSERT INTO sip_trunks (account_id, name, host, port, transport, from_user) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Demo Trunk', 'sip.example.com', 5060, 'udp', 'blayzen')
ON CONFLICT DO NOTHING;

-- =============================================================================
-- Verify Data
-- =============================================================================
DO $$
DECLARE
    account_count INT;
    route_count INT;
    trunk_count INT;
BEGIN
    SELECT COUNT(*) INTO account_count FROM accounts;
    SELECT COUNT(*) INTO route_count FROM sip_routes;
    SELECT COUNT(*) INTO trunk_count FROM sip_trunks;
    
    RAISE NOTICE 'Seed data loaded:';
    RAISE NOTICE '  Accounts: %', account_count;
    RAISE NOTICE '  Routes: %', route_count;
    RAISE NOTICE '  Trunks: %', trunk_count;
END $$;

