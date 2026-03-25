package push

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	"rustymanager/internal/db"
)

// Sender sends Web Push notifications to all stored subscriptions.
type Sender struct {
	queries         db.Querier
	vapidPublicKey  string
	vapidPrivateKey string
}

func NewSender(q db.Querier, pubKey, privKey string) *Sender {
	return &Sender{queries: q, vapidPublicKey: pubKey, vapidPrivateKey: privKey}
}

type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// Send fans out a push notification to all stored subscriptions.
// Stale subscriptions (410 Gone) are removed automatically.
func (s *Sender) Send(ctx context.Context, title, body, url string) {
	subs, err := s.queries.ListPushSubscriptions(ctx)
	if err != nil {
		log.Printf("push: list subscriptions: %v", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	p, err := json.Marshal(payload{Title: title, Body: body, URL: url})
	if err != nil {
		return
	}

	for _, sub := range subs {
		resp, err := webpush.SendNotification(p, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}, &webpush.Options{
			VAPIDPublicKey:  s.vapidPublicKey,
			VAPIDPrivateKey: s.vapidPrivateKey,
			TTL:             60,
		})
		if err != nil {
			log.Printf("push: send to %s: %v", sub.Endpoint, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			if delErr := s.queries.DeletePushSubscription(ctx, sub.Endpoint); delErr != nil {
				log.Printf("push: delete stale subscription: %v", delErr)
			}
		}
	}
}
