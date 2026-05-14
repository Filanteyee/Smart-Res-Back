package parking

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Seed fills parking_spots with test data only when the table is empty.
// Unlike sensors, parking statuses are preserved across restarts.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM parking_spots`).Scan(&count); err != nil {
		return fmt.Errorf("count: %w", err)
	}
	if count > 0 {
		return nil
	}
	for i := 1; i <= 20; i++ {
		num := fmt.Sprintf("P-%02d", i)
		if _, err := pool.Exec(ctx, `
			INSERT INTO parking_spots (spot_number, type) VALUES ($1, 'permanent')
			ON CONFLICT (spot_number) DO NOTHING`, num); err != nil {
			return fmt.Errorf("seed %s: %w", num, err)
		}
	}
	for i := 1; i <= 10; i++ {
		num := fmt.Sprintf("G-%02d", i)
		if _, err := pool.Exec(ctx, `
			INSERT INTO parking_spots (spot_number, type) VALUES ($1, 'guest')
			ON CONFLICT (spot_number) DO NOTHING`, num); err != nil {
			return fmt.Errorf("seed %s: %w", num, err)
		}
	}
	return nil
}
