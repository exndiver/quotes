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
		alertsWorkerLog().Printf("workers are disabled")
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
	alertsWorkerLog().Printf("workers started schedule=%s threshold=%s batch schedule=%d threshold=%d",
		worker.scheduleInterval,
		worker.thresholdInterval,
		worker.scheduleBatchSize,
		worker.thresholdBatchSize,
	)
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
		alertsWorkerLog().Printf("schedule find failed: %v", err)
		return
	}

	for _, dueAlert := range alerts {
		claimedAlert, err := w.repo.ClaimScheduleAlert(ctx, dueAlert, now)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				continue
			}
			alertsWorkerLog().Printf("schedule claim failed alert=%s device=%s err=%v", dueAlert.ID.Hex(), dueAlert.DeviceID, err)
			continue
		}

		alertsWorkerLog().Printf("schedule claimed alert=%s device=%s schedule_type=%s next_run_at=%v",
			claimedAlert.ID.Hex(),
			claimedAlert.DeviceID,
			claimedAlert.ScheduleType,
			claimedAlert.NextRunAt,
		)

		title, body := buildSchedulePushMessage(*claimedAlert)
		if err := w.sender.SendPush(ctx, claimedAlert.DeviceID, title, body); err != nil {
			alertsWorkerLog().Printf("schedule push failed alert=%s device=%s err=%v", claimedAlert.ID.Hex(), claimedAlert.DeviceID, err)
			continue
		}
		alertsWorkerLog().Printf("schedule push sent alert=%s device=%s", claimedAlert.ID.Hex(), claimedAlert.DeviceID)

		if err := w.repo.MarkAlertSent(ctx, claimedAlert.ID, time.Now().UTC()); err != nil {
			alertsWorkerLog().Printf("schedule mark sent failed alert=%s device=%s err=%v", claimedAlert.ID.Hex(), claimedAlert.DeviceID, err)
		}
	}
}

func (w *AlertWorker) runThreshold(ctx context.Context) {
	alerts, err := w.repo.GetActiveThresholdAlerts(ctx, w.thresholdBatchSize)
	if err != nil {
		alertsWorkerLog().Printf("threshold find failed: %v", err)
		return
	}

	for _, alert := range alerts {
		currentRate, err := CalculateCurrentRate(alert.Base, alert.Target)
		if err != nil {
			alertsWorkerLog().Printf("threshold rate failed alert=%s device=%s pair=%s err=%v", alert.ID.Hex(), alert.DeviceID, alert.Pair, err)
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
			alertsWorkerLog().Printf("threshold trigger update failed alert=%s device=%s err=%v", alert.ID.Hex(), alert.DeviceID, err)
			continue
		}

		alertsWorkerLog().Printf("threshold triggered alert=%s device=%s pair=%s value=%.6f current=%.6f direction=%s",
			triggeredAlert.ID.Hex(),
			triggeredAlert.DeviceID,
			triggeredAlert.Pair,
			triggeredAlert.Value,
			currentRate,
			triggeredAlert.Direction,
		)

		title, body := buildThresholdPushMessage(*triggeredAlert, currentRate)
		if err := w.sender.SendPush(ctx, triggeredAlert.DeviceID, title, body); err != nil {
			alertsWorkerLog().Printf("threshold push failed alert=%s device=%s err=%v", triggeredAlert.ID.Hex(), triggeredAlert.DeviceID, err)
			continue
		}
		alertsWorkerLog().Printf("threshold push sent alert=%s device=%s", triggeredAlert.ID.Hex(), triggeredAlert.DeviceID)

		if err := w.repo.MarkAlertSent(ctx, triggeredAlert.ID, time.Now().UTC()); err != nil {
			alertsWorkerLog().Printf("threshold mark sent failed alert=%s device=%s err=%v", triggeredAlert.ID.Hex(), triggeredAlert.DeviceID, err)
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
	pair := fmt.Sprintf("%s/%s", alert.Base, alert.Target)
	rate, err := CalculateCurrentRate(alert.Base, alert.Target)
	if err != nil {
		if alert.ScheduleType == AlertScheduleOnce {
			return "Quick update ⚡", fmt.Sprintf("%s right now: N/A", pair)
		}
		return "Staying updated 👀", fmt.Sprintf("%s is N/A", pair)
	}
	if alert.ScheduleType == AlertScheduleOnce {
		return "Quick update ⚡", fmt.Sprintf("%s right now: %s", pair, formatAlertRateForPush(alert.Base, alert.Target, rate))
	}
	return "Staying updated 👀", fmt.Sprintf("%s is %s", pair, formatAlertRateForPush(alert.Base, alert.Target, rate))
}

func buildThresholdPushMessage(alert Alert, currentRate float64) (string, string) {
	pair := fmt.Sprintf("%s/%s", alert.Base, alert.Target)
	if alert.Direction == AlertDirectionDown {
		return "Target hit 🎯", fmt.Sprintf("%s dropped to %s", pair, formatAlertRateForPush(alert.Base, alert.Target, currentRate))
	}
	return "Target hit 🎯", fmt.Sprintf("%s climbed to %s", pair, formatAlertRateForPush(alert.Base, alert.Target, currentRate))
}
