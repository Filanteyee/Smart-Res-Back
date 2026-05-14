ALTER TABLE barrier_events ADD COLUMN IF NOT EXISTS gate_id VARCHAR(50) NOT NULL DEFAULT 'main-gate';

CREATE INDEX IF NOT EXISTS idx_barrier_events_gate ON barrier_events(gate_id);
