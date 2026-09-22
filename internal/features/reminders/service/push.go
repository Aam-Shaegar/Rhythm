package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/SherClockHolmes/webpush-go"
)

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// 404/410 от push-сервиса — подписку удалить.
var ErrSubscriptionGone = errors.New("push subscription gone")

type PushSender interface {
	Send(ctx context.Context, sub *domain.PushSubscription, payload []byte) error
}

type WebPushSender struct {
	PublicKey  string
	privateKey string
	subject    string
	ttl        int
}

func NewWebPushSender(publicKey, privateKey, subject string) *WebPushSender {
	// webpush-go сам добавляет "mailto:", иначе Apple отвечает 403 BadJwtToken.
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "admin@rhythm.local"
	} else {
		subject = strings.TrimPrefix(subject, "mailto:")
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
		// Тело ответа содержит реальную причину (Apple/Google).
		if resp != nil {
			body := drainPushRespBody(resp)
			if resp.StatusCode == 404 || resp.StatusCode == 410 {
				return ErrSubscriptionGone
			}
			if body != "" {
				return fmt.Errorf("webpush send: %w (status %d: %s)", err, resp.StatusCode, body)
			}
		}
		return fmt.Errorf("webpush send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return ErrSubscriptionGone
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webpush unexpected status: %d: %s", resp.StatusCode, drainPushRespBody(resp))
	}
	return nil
}

// Тело ответа — только короткая причина, без endpoint/ключей.
func drainPushRespBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil || len(b) == 0 {
		return ""
	}
	return strings.TrimSpace(string(b))
}

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
