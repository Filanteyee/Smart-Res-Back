CREATE TABLE IF NOT EXISTS parking_spots (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    spot_number      VARCHAR(20) NOT NULL UNIQUE,
    type             VARCHAR(10) NOT NULL CHECK (type IN ('permanent', 'guest')),
    status           VARCHAR(10) NOT NULL DEFAULT 'free' CHECK (status IN ('free', 'occupied', 'reserved')),
    assigned_user_id UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_parking_spots_user   ON parking_spots(assigned_user_id);
CREATE INDEX IF NOT EXISTS idx_parking_spots_status ON parking_spots(status);

CREATE TABLE IF NOT EXISTS parking_bookings (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    spot_id    UUID        NOT NULL REFERENCES parking_spots(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)          ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time   TIMESTAMPTZ NOT NULL,
    status     VARCHAR(10) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_parking_bookings_user ON parking_bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_parking_bookings_spot ON parking_bookings(spot_id, status);

CREATE TABLE IF NOT EXISTS parking_events (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    spot_id    UUID        NOT NULL REFERENCES parking_spots(id) ON DELETE CASCADE,
    event_type VARCHAR(10) NOT NULL CHECK (event_type IN ('occupied', 'freed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
