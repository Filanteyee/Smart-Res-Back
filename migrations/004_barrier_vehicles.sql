CREATE TABLE IF NOT EXISTS vehicles (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plate_number VARCHAR(20)  NOT NULL,
    brand        VARCHAR(100) NOT NULL DEFAULT '',
    color        VARCHAR(50)  NOT NULL DEFAULT '',
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vehicles_plate ON vehicles(plate_number);
CREATE INDEX        IF NOT EXISTS idx_vehicles_user  ON vehicles(user_id);

CREATE TABLE IF NOT EXISTS barrier_events (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type    VARCHAR(20)  NOT NULL,
    direction     VARCHAR(5),
    plate_number  VARCHAR(20),
    vehicle_id    UUID REFERENCES vehicles(id)      ON DELETE SET NULL,
    guest_pass_id UUID REFERENCES guest_access(id)  ON DELETE SET NULL,
    opened_by     UUID REFERENCES users(id)          ON DELETE SET NULL,
    status        VARCHAR(10)  NOT NULL DEFAULT 'OPENED',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_barrier_events_created ON barrier_events(created_at DESC);

ALTER TABLE guest_access ADD COLUMN IF NOT EXISTS qr_code VARCHAR(36);

CREATE UNIQUE INDEX IF NOT EXISTS idx_guest_access_qr
    ON guest_access(qr_code) WHERE qr_code IS NOT NULL;
