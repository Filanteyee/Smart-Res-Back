package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notifier interface {
	NotifyEvent(ctx context.Context, eventID string) (int, error)
}

// Broadcaster pushes SSE frames to connected admins. Optional — may be nil.
type Broadcaster interface {
	Broadcast(event string, data any) error
}

type Config struct {
	URL      string
	Username string
	Password string
	ClientID string
}

type Subscriber struct {
	db       *pgxpool.Pool
	notifier Notifier
	bcast    Broadcaster
	client   paho.Client
}

func New(cfg Config, db *pgxpool.Pool, notifier Notifier, bcast Broadcaster) (*Subscriber, error) {
	s := &Subscriber{db: db, notifier: notifier, bcast: bcast}

	opts := paho.NewClientOptions()
	opts.AddBroker(cfg.URL)
	opts.SetClientID(cfg.ClientID)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetCleanSession(true)
	opts.SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12})

	opts.SetOnConnectHandler(func(c paho.Client) {
		log.Println("[mqtt] connected")
		t := c.Subscribe("smartresidency/sensors/+/+/+", 1, s.handleSensor)
		t.Wait()
		if err := t.Error(); err != nil {
			log.Printf("[mqtt] subscribe sensors FAILED: %v", err)
		} else {
			log.Println("[mqtt] subscribed to smartresidency/sensors/+/+/+ (QoS 1)")
		}
		t2 := c.Subscribe("smartresidency/events/+", 1, s.handleEvent)
		t2.Wait()
		if err := t2.Error(); err != nil {
			log.Printf("[mqtt] subscribe events FAILED: %v", err)
		} else {
			log.Println("[mqtt] subscribed to smartresidency/events/+ (QoS 1)")
		}
	})
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		log.Printf("[mqtt] disconnected: %v", err)
	})

	s.client = paho.NewClient(opts)
	if t := s.client.Connect(); t.Wait() && t.Error() != nil {
		return nil, t.Error()
	}
	return s, nil
}

type sensorMsg struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	EntranceNum int    `json:"entrance_num"`
	Floor       int    `json:"floor"`
	Status      string `json:"status"`
}

type eventMsg struct {
	SensorID    string `json:"sensor_id"`
	Type        string `json:"type"`
	EntranceNum int    `json:"entrance_num"`
	Floor       int    `json:"floor"`
	Status      string `json:"status"`
}

func (s *Subscriber) handleSensor(_ paho.Client, m paho.Message) {
	log.Printf("[mqtt] RX %s: %s", m.Topic(), string(m.Payload()))
	var msg sensorMsg
	if err := json.Unmarshal(m.Payload(), &msg); err != nil {
		log.Printf("[mqtt] sensor payload: %v", err)
		return
	}
	if msg.ID == "" || msg.Status == "" {
		log.Printf("[mqtt] sensor payload: missing id/status")
		return
	}
	ctx := context.Background()

	var prev string
	hadRow := s.db.QueryRow(ctx, `SELECT status FROM sensors WHERE id=$1`, msg.ID).Scan(&prev) == nil

	// Upsert. last_seen_at = NOW() on EVERY message including heartbeat pings,
	// so the OFFLINE sweeper can tell live from dead sensors.
	_, err := s.db.Exec(ctx, `
		INSERT INTO sensors (id, type, entrance_num, floor, status, last_updated, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE
		SET status = EXCLUDED.status,
		    last_updated = NOW(),
		    last_seen_at = NOW()`,
		msg.ID, msg.Type, msg.EntranceNum, msg.Floor, msg.Status,
	)
	if err != nil {
		log.Printf("[mqtt] sensor upsert %s: %v", msg.ID, err)
		return
	}

	// Broadcast sensor_update only on status changes — heartbeats (status
	// unchanged) would otherwise flood the SSE channel 54 times every 30 sec.
	if s.bcast != nil && (!hadRow || prev != msg.Status) {
		_ = s.bcast.Broadcast("sensor_update", map[string]any{
			"id":           msg.ID,
			"type":         msg.Type,
			"entrance_num": msg.EntranceNum,
			"floor":        msg.Floor,
			"status":       msg.Status,
		})
	}

	// Event creation rules:
	//   NORMAL→ALERT   : new event, push residents (alarm raised).
	//   OFFLINE→ALERT  : new event, push residents (sensor came back already firing).
	//   OFFLINE→NORMAL : silent recovery, no event, no push.
	//   NORMAL→NORMAL  : heartbeat, only last_seen_at is touched (above).
	//   ALERT→NORMAL   : sensor cleared itself / reset endpoint, no push.
	//   ALERT→ALERT    : duplicate, ignore.
	shouldCreateEvent := hadRow && msg.Status == "ALERT" && (prev == "NORMAL" || prev == "OFFLINE")
	if shouldCreateEvent {
		var eventID string
		err := s.db.QueryRow(ctx, `
			INSERT INTO sensor_events (id, sensor_id, type, entrance_num, floor, status)
			VALUES ('evt-' || nextval('sensor_events_seq'), $1, $2, $3, $4, 'DETECTED')
			RETURNING id`,
			msg.ID, msg.Type, msg.EntranceNum, msg.Floor,
		).Scan(&eventID)
		if err != nil {
			log.Printf("[mqtt] event insert: %v", err)
			return
		}
		log.Printf("[mqtt] %s: %s→ALERT, event %s", msg.ID, prev, eventID)
		if s.bcast != nil {
			_ = s.bcast.Broadcast("event_new", map[string]any{
				"id":           eventID,
				"sensor_id":    msg.ID,
				"type":         msg.Type,
				"entrance_num": msg.EntranceNum,
				"floor":        msg.Floor,
				"status":       "DETECTED",
			})
		}
		if s.notifier != nil {
			go func(id string) {
				if _, err := s.notifier.NotifyEvent(context.Background(), id); err != nil {
					log.Printf("[mqtt] notify %s: %v", id, err)
				}
			}(eventID)
		}
	}
}

func (s *Subscriber) handleEvent(_ paho.Client, m paho.Message) {
	var msg eventMsg
	if err := json.Unmarshal(m.Payload(), &msg); err != nil {
		log.Printf("[mqtt] event payload: %v", err)
		return
	}
	if msg.SensorID == "" {
		log.Printf("[mqtt] event payload: missing sensor_id")
		return
	}
	status := msg.Status
	if status == "" {
		status = "DETECTED"
	}

	var eventID string
	err := s.db.QueryRow(context.Background(), `
		INSERT INTO sensor_events (id, sensor_id, type, entrance_num, floor, status)
		VALUES ('evt-' || nextval('sensor_events_seq'), $1, $2, $3, $4, $5)
		RETURNING id`,
		msg.SensorID, msg.Type, msg.EntranceNum, msg.Floor, status,
	).Scan(&eventID)
	if err != nil {
		log.Printf("[mqtt] event insert: %v", err)
		return
	}
	log.Printf("[mqtt] event %s from %s", eventID, msg.SensorID)
	if s.notifier != nil {
		go func(id string) {
			if _, err := s.notifier.NotifyEvent(context.Background(), id); err != nil {
				log.Printf("[mqtt] notify %s: %v", id, err)
			}
		}(eventID)
	}
}

// Publish sends a JSON payload to the broker. retain=true so subscribers
// (and the broker itself across restarts) keep the last value per topic.
// Used by POST /admin/sensors/:id/reset to issue a status=NORMAL command
// that the same broker echoes back to our subscriber.
func (s *Subscriber) Publish(topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	t := s.client.Publish(topic, 1, true, data)
	t.Wait()
	return t.Error()
}

func (s *Subscriber) Close() {
	if s.client != nil {
		s.client.Disconnect(250)
	}
}
