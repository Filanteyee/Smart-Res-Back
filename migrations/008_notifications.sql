CREATE TABLE IF NOT EXISTS notifications (
    id             UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_user_id UUID        REFERENCES users(id) ON DELETE CASCADE,
    target_role    VARCHAR(50),
    kind           VARCHAR(50) NOT NULL,
    title          TEXT        NOT NULL,
    body           TEXT        NOT NULL,
    data           JSONB,
    read_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user   ON notifications(target_user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_role   ON notifications(target_role);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(created_at DESC) WHERE read_at IS NULL;
