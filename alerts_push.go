package main

import (
	"context"
	"errors"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type PushSender interface {
	SendPush(ctx context.Context, deviceID, title, body string) error
}

type FirebasePushSender struct {
	repo   *AlertRepository
	client *messaging.Client
}

func NewFirebasePushSender(ctx context.Context, repo *AlertRepository, credentialsFile string) (*FirebasePushSender, error) {
	if credentialsFile == "" {
		return nil, errors.New("firebase credentials_file is empty")
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &FirebasePushSender{
		repo:   repo,
		client: client,
	}, nil
}

func (s *FirebasePushSender) SendPush(ctx context.Context, deviceID, title, body string) error {
	device, err := s.repo.GetDevice(ctx, deviceID)
	if err != nil {
		alertsWorkerLog().Printf("push: device lookup failed device=%s err=%v", deviceID, err)
		return err
	}

	message := &messaging.Message{
		Token: device.PushToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}

	_, err = s.client.Send(ctx, message)
	if err != nil {
		if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
			alertsWorkerLog().Printf("push: deactivating device=%s due to token error: %v", deviceID, err)
			_ = s.repo.DeactivateDevice(ctx, deviceID)
		}
		alertsWorkerLog().Printf("push: send failed device=%s err=%v", deviceID, err)
		return err
	}

	alertsWorkerLog().Printf("push: send ok device=%s", deviceID)
	return nil
}
