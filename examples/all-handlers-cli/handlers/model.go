package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/transire/transire"
)

const (
	WorkQueue          = "work-events"
	NotificationsQueue = "notifications"
	NotificationLog    = "notification-log"
)

type WorkPayload struct {
	Source string `json:"source"`
	Detail string `json:"detail"`
}

type NotificationPayload struct {
	Stage  string `json:"stage"`
	Detail string `json:"detail"`
}

func sendWork(ctx context.Context, sender transire.QueueSender, payload WorkPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal work payload: %w", err)
	}
	return sender.Send(ctx, WorkQueue, body)
}

func sendNotification(ctx context.Context, sender transire.QueueSender, payload NotificationPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	return sender.Send(ctx, NotificationsQueue, body)
}
