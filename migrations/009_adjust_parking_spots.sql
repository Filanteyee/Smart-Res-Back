-- Permanent spots: expand to 20 (add P-11..P-20 if missing).
INSERT INTO parking_spots (spot_number, type)
SELECT 'P-' || LPAD(g::text, 2, '0'), 'permanent'
FROM generate_series(11, 20) g
ON CONFLICT (spot_number) DO NOTHING;

-- Guest spots: reduce to 10 (remove G-11..G-20 that have no active bookings).
DELETE FROM parking_spots
WHERE spot_number LIKE 'G-%'
  AND CAST(SUBSTRING(spot_number FROM 3) AS INTEGER) > 10
  AND assigned_user_id IS NULL
  AND id NOT IN (
      SELECT spot_id FROM parking_bookings
      WHERE status IN ('active', 'upcoming')
  );
