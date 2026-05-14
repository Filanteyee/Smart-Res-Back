-- Heartbeat / OFFLINE detection + sensor_event timeline columns + audit.
-- Applied 2026-05-11. See cmd/server/main.go startup log to confirm.

-- last_seen_at: updated on EVERY sensor MQTT message (including heartbeat
-- pings every 30 sec from Node-RED). Used by the OFFLINE sweeper to mark
-- sensors as OFFLINE if no message arrived in >60 sec.
ALTER TABLE sensors
    ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Timeline columns on sensor_events so GET /sensors/events/:id can render
-- DETECTED → CHECKING → CONFIRMED/FALSE_ALARM transitions without a
-- separate audit table. Each column captures when that transition happened.
ALTER TABLE sensor_events
    ADD COLUMN IF NOT EXISTS checking_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS false_alarmed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS confirmed_by     UUID REFERENCES users(id);

-- Backfill: sensors created before this migration get last_seen_at = NOW(),
-- so the sweeper does not immediately mark them OFFLINE on first run.
UPDATE sensors SET last_seen_at = NOW() WHERE last_seen_at IS NULL;
