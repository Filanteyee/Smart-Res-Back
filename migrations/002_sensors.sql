-- IoT subsystem: water/smoke sensors, events, FCM tokens.
-- Adds entrance/floor/apartment to profiles so backend can map a resident
-- to their physical entrance for targeted push delivery.

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS entrance  INT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS floor     INT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS apartment VARCHAR(50) DEFAULT '';

CREATE TABLE IF NOT EXISTS sensors (
    id           TEXT PRIMARY KEY,
    type         TEXT NOT NULL,
    entrance_num INT  NOT NULL,
    floor        INT  NOT NULL,
    status       TEXT NOT NULL DEFAULT 'NORMAL',
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sensors_entrance ON sensors(entrance_num);

CREATE SEQUENCE IF NOT EXISTS sensor_events_seq START 1;

CREATE TABLE IF NOT EXISTS sensor_events (
    id            TEXT PRIMARY KEY,
    sensor_id     TEXT NOT NULL REFERENCES sensors(id),
    type          TEXT NOT NULL,
    entrance_num  INT  NOT NULL,
    floor         INT  NOT NULL,
    status        TEXT NOT NULL DEFAULT 'DETECTED',
    threat_type   TEXT,
    admin_comment TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sensor_events_created  ON sensor_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sensor_events_entrance ON sensor_events(entrance_num);
CREATE INDEX IF NOT EXISTS idx_sensor_events_status   ON sensor_events(status);

CREATE TABLE IF NOT EXISTS fcm_tokens (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT PRIMARY KEY,
    platform   TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fcm_user ON fcm_tokens(user_id);
