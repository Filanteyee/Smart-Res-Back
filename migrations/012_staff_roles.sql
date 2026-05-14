-- Staff roles: add specialty to profiles + assignment tracking to service_requests.
ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS specialty VARCHAR(50);

ALTER TABLE service_requests
    ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS taken_at    TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_sr_assigned ON service_requests(assigned_to);
