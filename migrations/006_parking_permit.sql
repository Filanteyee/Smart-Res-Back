ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS parking_permit_status VARCHAR(10) NOT NULL DEFAULT 'none'
        CHECK (parking_permit_status IN ('none', 'pending', 'approved', 'rejected'));

CREATE TABLE IF NOT EXISTS parking_permit_requests (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vehicle_plate VARCHAR(20) NOT NULL,
    comment       TEXT        NOT NULL DEFAULT '',
    status        VARCHAR(10) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'approved', 'rejected')),
    admin_comment TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_parking_permit_requests_user   ON parking_permit_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_parking_permit_requests_status ON parking_permit_requests(status);
