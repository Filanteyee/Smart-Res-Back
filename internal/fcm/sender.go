package fcm

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/option"
)

type Sender struct {
	db        *pgxpool.Pool
	messaging *messaging.Client
}

func New(ctx context.Context, credPath string, db *pgxpool.Pool) (*Sender, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, fmt.Errorf("firebase init: %w", err)
	}
	msg, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase messaging: %w", err)
	}
	return &Sender{db: db, messaging: msg}, nil
}

// NotifyEvent loads the event, finds approved residents of its entrance,
// sends an FCM data-message to all their tokens, and prunes invalid ones.
func (s *Sender) NotifyEvent(ctx context.Context, eventID string) (int, error) {
	var (
		sensorID, typ, status string
		threat                *string
		entrance, floor       int
	)
	err := s.db.QueryRow(ctx, `
		SELECT sensor_id, type, status, threat_type, entrance_num, floor
		FROM sensor_events WHERE id = $1`, eventID,
	).Scan(&sensorID, &typ, &status, &threat, &entrance, &floor)
	if err != nil {
		return 0, fmt.Errorf("load event: %w", err)
	}

	rows, err := s.db.Query(ctx, `
		SELECT t.token FROM fcm_tokens t
		JOIN profiles p ON p.id = t.user_id
		WHERE p.entrance = $1 AND p.verification_status = 'approved'`, entrance)
	if err != nil {
		return 0, fmt.Errorf("load tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			tokens = append(tokens, t)
		}
	}
	if len(tokens) == 0 {
		log.Printf("[fcm] event %s: no tokens for entrance %d", eventID, entrance)
		return 0, nil
	}

	threatStr := ""
	if threat != nil {
		threatStr = *threat
	}
	title, body := messageFor(typ, status, threatStr, entrance, floor)

	data := map[string]string{
		"kind":         "sensor_alert",
		"event_id":     eventID,
		"threat_type":  threatStr,
		"entrance_num": fmt.Sprintf("%d", entrance),
		"floor":        fmt.Sprintf("%d", floor),
		"status":       status,
		"title":        title,
		"body":         body,
	}

	multicast := &messaging.MulticastMessage{
		Tokens:  tokens,
		Data:    data,
		Android: &messaging.AndroidConfig{Priority: "high"},
	}

	resp, err := s.messaging.SendEachForMulticast(ctx, multicast)
	if err != nil {
		return 0, fmt.Errorf("send: %w", err)
	}
	log.Printf("[fcm] event %s: success=%d failure=%d", eventID, resp.SuccessCount, resp.FailureCount)

	if resp.FailureCount > 0 {
		for i, r := range resp.Responses {
			if r.Success {
				continue
			}
			log.Printf("[fcm] token[%d] error: %v", i, r.Error)
			if messaging.IsRegistrationTokenNotRegistered(r.Error) || messaging.IsUnregistered(r.Error) {
				log.Printf("[fcm] pruning unregistered token: %s...", tokens[i][:20])
				_, _ = s.db.Exec(ctx, `DELETE FROM fcm_tokens WHERE token = $1`, tokens[i])
			}
		}
	}
	return resp.SuccessCount, nil
}

func messageFor(typ, status, threat string, entrance, floor int) (string, string) {
	isFire := typ == "SMOKE" || threat == "FIRE"
	isWater := typ == "WATER" || threat == "WATER_LEAK"

	switch {
	case status == "CONFIRMED" && isFire:
		return "Тревога: пожар",
			fmt.Sprintf("%d подъезд, этаж %d. Проверка подтверждена администратором.", entrance, floor)
	case status == "CONFIRMED" && isWater:
		return "Тревога: затопление",
			fmt.Sprintf("%d подъезд, этаж %d. Проверка подтверждена администратором.", entrance, floor)
	case isFire:
		return "Тревога: возможное задымление",
			fmt.Sprintf("%d подъезд, этаж %d. Сработал датчик дыма.", entrance, floor)
	case isWater:
		return "Тревога: возможное затопление",
			fmt.Sprintf("%d подъезд, этаж %d. Сработал датчик воды.", entrance, floor)
	default:
		return "Тревога", fmt.Sprintf("%d подъезд, этаж %d.", entrance, floor)
	}
}
