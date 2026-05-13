package sensors

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	Entrances         = 3
	FloorsPerEntrance = 9
)

var sensorTypes = []struct {
	prefix string
	kind   string
}{
	{"w", "WATER"},
	{"s", "SMOKE"},
}

// Seed creates the sensor registry on first start and resets statuses on each
// restart. Reset on restart is intentional: this is a simulation environment
// where Node-RED replays sensor state, so we always start from NORMAL.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	for e := 1; e <= Entrances; e++ {
		for f := 1; f <= FloorsPerEntrance; f++ {
			for _, t := range sensorTypes {
				id := fmt.Sprintf("%s-%d-%d", t.prefix, e, f)
				_, err := pool.Exec(ctx, `
					INSERT INTO sensors (id, type, entrance_num, floor, status, last_updated, last_seen_at)
					VALUES ($1, $2, $3, $4, 'NORMAL', NOW(), NOW())
					ON CONFLICT (id) DO UPDATE
					SET status = 'NORMAL', last_updated = NOW(), last_seen_at = NOW()`,
					id, t.kind, e, f,
				)
				if err != nil {
					return fmt.Errorf("seed %s: %w", id, err)
				}
			}
		}
	}
	return nil
}
