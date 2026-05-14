-- Новая таблица parking_permits: permit привязан к конкретному автомобилю жильца.
-- Право на въезд = vehicle + approved parking_permit (вместо profile.parking_permit_status).
CREATE TABLE IF NOT EXISTS parking_permits (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID        NOT NULL REFERENCES users(id)          ON DELETE CASCADE,
    vehicle_id    UUID        NOT NULL REFERENCES vehicles(id)        ON DELETE CASCADE,
    spot_id       UUID                 REFERENCES parking_spots(id)  ON DELETE SET NULL,
    status        VARCHAR(10) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'approved', 'rejected')),
    document_url  TEXT,
    admin_comment TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at   TIMESTAMPTZ
);

-- На одну машину может быть только один одобренный пропуск
CREATE UNIQUE INDEX IF NOT EXISTS idx_parking_permits_vehicle_approved
    ON parking_permits(vehicle_id) WHERE status = 'approved';

CREATE INDEX IF NOT EXISTS idx_parking_permits_user   ON parking_permits(user_id);
CREATE INDEX IF NOT EXISTS idx_parking_permits_status ON parking_permits(status);
