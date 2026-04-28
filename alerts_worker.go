package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type AlertWorker struct {
	repo               *AlertRepository
	sender             PushSender
	scheduleInterval   time.Duration
	thresholdInterval  time.Duration
	scheduleBatchSize  int64
	thresholdBatchSize int64
}

func StartAlertWorkers(ctx context.Context) error {
	if !Config.AlertsWorkers.Enabled {
		fmt.Printf("Alert workers are disabled\n")
		return nil
	}

	repo := NewAlertRepository(client)
	sender, err := NewFirebasePushSender(ctx, repo, Config.Firebase.CredentialsFile)
	if err != nil {
		return fmt.Errorf("firebase push sender init failed: %w", err)
	}

	worker := &AlertWorker{
		repo:               repo,
		sender:             sender,
		scheduleInterval:   parseAlertWorkerInterval(Config.AlertsWorkers.ScheduleInterval, 5*time.Minute),
		thresholdInterval:  parseAlertWorkerInterval(Config.AlertsWorkers.ThresholdInterval, 30*time.Second),
		scheduleBatchSize:  int64(defaultPositiveInt(Config.AlertsWorkers.ScheduleBatchSize, 50)),
		thresholdBatchSize: int64(defaultPositiveInt(Config.AlertsWorkers.ThresholdBatchSize, 100)),
	}

	go worker.scheduleLoop(ctx)
	go worker.thresholdLoop(ctx)
	fmt.Printf("Alert workers started: schedule=%s threshold=%s\n", worker.scheduleInterval, worker.thresholdInterval)
	return nil
}

func parseAlertWorkerInterval(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return fallback
	}
	return duration
}

func defaultPositiveInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func (w *AlertWorker) scheduleLoop(ctx context.Context) {
	w.runSchedule(ctx)
	ticker := time.NewTicker(w.scheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runSchedule(ctx)
		}
	}
}

func (w *AlertWorker) thresholdLoop(ctx context.Context) {
	w.runThreshold(ctx)
	ticker := time.NewTicker(w.thresholdInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runThreshold(ctx)
		}
	}
}

func (w *AlertWorker) runSchedule(ctx context.Context) {
	now := time.Now().UTC()
	alerts, err := w.repo.FindDueScheduleAlerts(ctx, now, w.scheduleBatchSize)
	if err != nil {
		fmt.Printf("Schedule worker find failed: %v\n", err)
		return
	}

	for _, dueAlert := range alerts {
		claimedAlert, err := w.repo.ClaimScheduleAlert(ctx, dueAlert, now)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				continue
			}
			fmt.Printf("Schedule worker claim failed: %v\n", err)
			continue
		}

		title, body := buildSchedulePushMessage(*claimedAlert)
		if err := w.sender.SendPush(ctx, claimedAlert.DeviceID, title, body); err != nil {
			fmt.Printf("Schedule push failed for alert %s: %v\n", claimedAlert.ID.Hex(), err)
			continue
		}

		if err := w.repo.MarkAlertSent(ctx, claimedAlert.ID, time.Now().UTC()); err != nil {
			fmt.Printf("Schedule worker mark sent failed for alert %s: %v\n", claimedAlert.ID.Hex(), err)
		}
	}
}

func (w *AlertWorker) runThreshold(ctx context.Context) {
	alerts, err := w.repo.GetActiveThresholdAlerts(ctx, w.thresholdBatchSize)
	if err != nil {
		fmt.Printf("Threshold worker find failed: %v\n", err)
		return
	}

	for _, alert := range alerts {
		currentRate, err := CalculateCurrentRate(alert.Base, alert.Target)
		if err != nil {
			fmt.Printf("Threshold worker rate failed for alert %s: %v\n", alert.ID.Hex(), err)
			continue
		}
		if !isThresholdTriggered(alert, currentRate) {
			continue
		}

		triggeredAlert, err := w.repo.MarkAlertTriggered(ctx, alert.ID, currentRate)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				continue
			}
			fmt.Printf("Threshold worker trigger failed: %v\n", err)
			continue
		}

		title, body := buildThresholdPushMessage(*triggeredAlert, currentRate)
		if err := w.sender.SendPush(ctx, triggeredAlert.DeviceID, title, body); err != nil {
			fmt.Printf("Threshold push failed for alert %s: %v\n", triggeredAlert.ID.Hex(), err)
			continue
		}

		if err := w.repo.MarkAlertSent(ctx, triggeredAlert.ID, time.Now().UTC()); err != nil {
			fmt.Printf("Threshold worker mark sent failed for alert %s: %v\n", triggeredAlert.ID.Hex(), err)
		}
	}
}

func isThresholdTriggered(alert Alert, currentRate float64) bool {
	switch alert.Direction {
	case AlertDirectionUp:
		return currentRate >= alert.Value
	case AlertDirectionDown:
		return currentRate <= alert.Value
	default:
		return false
	}
}

func buildSchedulePushMessage(alert Alert) (string, string) {
	title := fmt.Sprintf("%s/%s rate reminder", alert.Base, alert.Target)
	rate, err := CalculateCurrentRate(alert.Base, alert.Target)
	if err != nil {
		return title, "Your scheduled rate reminder is due."
	}
	return title, fmt.Sprintf("Current %s/%s rate is %.4f", alert.Base, alert.Target, rate)
}

func buildThresholdPushMessage(alert Alert, currentRate float64) (string, string) {
	title := fmt.Sprintf("%s/%s alert triggered", alert.Base, alert.Target)
	return title, fmt.Sprintf("Current rate %.4f reached your %.4f target.", currentRate, alert.Value)
}
