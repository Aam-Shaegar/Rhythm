package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/SherClockHolmes/webpush-go"
)

// pushPayload is what the service worker receives in the "push" event
// and renders via showNotification.
type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// ErrSubscriptionGone means the push service reports the endpoint as
// expired (HTTP 404/410) — the subscription must be deleted.
var ErrSubscriptionGone = errors.New("push subscription gone")

// PushSender delivers one payload to one subscription.
// Implementations: *WebPushSender (production), fakes in tests.
type PushSender interface {
	Send(ctx context.Context, sub *domain.PushSubscription, payload []byte) error
}

// WebPushSender sends RFC 8030 Web Push notifications.
type WebPushSender struct {
	PublicKey  string
	privateKey string
	subject    string
	ttl        int
}

func NewWebPushSender(publicKey, privateKey, subject string) *WebPushSender {
	if subject == "" {
		subject = "mailto:admin@rhythm.local"
	}
	return &WebPushSender{PublicKey: publicKey, privateKey: privateKey, subject: subject, ttl: 86400}
}

func (s *WebPushSender) Send(ctx context.Context, sub *domain.PushSubscription, payload []byte) error {
	resp, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256DH, Auth: sub.Auth},
	}, &webpush.Options{
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.PublicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             s.ttl,
	})
	if err != nil {
		// webpush-go may still return the response on protocol errors.
		if resp != nil && (resp.StatusCode == 404 || resp.StatusCode == 410) {
			return ErrSubscriptionGone
		}
		return fmt.Errorf("webpush send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return ErrSubscriptionGone
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webpush unexpected status: %d", resp.StatusCode)
	}
	return nil
}

// BuildPushPayload renders a reminder into a notification.
func BuildPushPayload(rem *domain.Reminder) []byte {
	title := rem.Title
	if title == "" {
		title = "Напоминание"
	}
	var body string
	switch rem.EntityType {
	case "event":
		body = fmt.Sprintf("Событие «%s» скоро начнётся", rem.Title)
	case "task":
		body = fmt.Sprintf("Приближается срок задачи «%s»", rem.Title)
	default:
		body = rem.Title
	}
	if rem.Title == "" {
		body = "У вас новое напоминание"
	}
	payload, _ := json.Marshal(pushPayload{Title: title, Body: body, URL: "/"})
	return payload
}
