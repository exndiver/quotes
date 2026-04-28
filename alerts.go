package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	alertsDBName          = "Quotes"
	devicesCollectionName = "devices"
	alertsCollectionName  = "alerts"
	AlertTypeThreshold    = "threshold"
	AlertTypeSchedule     = "schedule"
	AlertStatusActive     = "active"
	AlertStatusTriggered  = "triggered"
	AlertDirectionUp      = "up"
	AlertDirectionDown    = "down"
	AlertScheduleOnce     = "once"
	AlertScheduleWeekly   = "weekly"
	DevicePlatformIOS     = "ios"
	DevicePlatformAndroid = "android"
)

var (
	ErrInvalidDevice = errors.New("invalid device")
	ErrInvalidAlert  = errors.New("invalid alert")
	ErrAlertNotFound = errors.New("alert not found")
	ErrRateNotFound  = errors.New("rate not found")
)

// Device is a push notification receiver. Mongo _id is the client device_id.
type Device struct {
	ID         string    `bson:"_id,omitempty" json:"device_id"`
	PushToken  string    `bson:"push_token" json:"push_token"`
	Platform   string    `bson:"platform" json:"platform"`
	IsActive   bool      `bson:"is_active" json:"is_active"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
	LastSeenAt time.Time `bson:"last_seen_at" json:"last_seen_at"`
}

// Alert describes one notification rule for a currency pair.
type Alert struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	DeviceID     string             `bson:"device_id" json:"device_id"`
	Type         string             `bson:"type" json:"type"`
	Base         string             `bson:"base" json:"base"`
	Target       string             `bson:"target" json:"target"`
	Pair         string             `bson:"pair" json:"pair"`
	Status       string             `bson:"status" json:"status"`
	Value        float64            `bson:"value,omitempty" json:"value,omitempty"`
	Direction    string             `bson:"direction,omitempty" json:"direction,omitempty"`
	ScheduleType string             `bson:"schedule_type,omitempty" json:"schedule_type,omitempty"`
	ScheduledAt  *time.Time         `bson:"scheduled_at,omitempty" json:"scheduled_at,omitempty"`
	DaysOfWeek   []int              `bson:"days_of_week,omitempty" json:"days_of_week,omitempty"`
	Hour         *int               `bson:"hour,omitempty" json:"hour,omitempty"`
	Timezone     string             `bson:"timezone,omitempty" json:"timezone,omitempty"`
	NextRunAt    *time.Time         `bson:"next_run_at,omitempty" json:"next_run_at,omitempty"`
	LastSentAt   *time.Time         `bson:"last_sent_at,omitempty" json:"last_sent_at,omitempty"`
	TriggeredAt  *time.Time         `bson:"triggered_at,omitempty" json:"triggered_at,omitempty"`
	TriggerRate  float64            `bson:"trigger_rate,omitempty" json:"trigger_rate,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

type AlertRepository struct {
	client *mongo.Client
	dbName string
}

func NewAlertRepository(client *mongo.Client) *AlertRepository {
	return &AlertRepository{
		client: client,
		dbName: alertsDBName,
	}
}

func (r *AlertRepository) devices() *mongo.Collection {
	return r.client.Database(r.dbName).Collection(devicesCollectionName)
}

func (r *AlertRepository) alerts() *mongo.Collection {
	return r.client.Database(r.dbName).Collection(alertsCollectionName)
}

func BuildAlertPair(base, target string) string {
	return fmt.Sprintf("%s_%s", strings.ToUpper(base), strings.ToUpper(target))
}

func CalculateCurrentRate(base, target string) (float64, error) {
	base = strings.ToUpper(base)
	target = strings.ToUpper(target)
	if base == "" || target == "" {
		return 0, ErrRateNotFound
	}
	if base == target {
		if _, ok := getRateFromDB(base, 0); !ok {
			return 0, ErrRateNotFound
		}
		return 1, nil
	}

	baseRate, ok := getRateFromDB(base, 0)
	if !ok || baseRate == 0 {
		return 0, ErrRateNotFound
	}
	targetRate, ok := getRateFromDB(target, 0)
	if !ok {
		return 0, ErrRateNotFound
	}

	return math.Round((targetRate/baseRate)*10000000000) / 10000000000, nil
}

func CalculateNextRunAt(scheduleType string, scheduledAt *time.Time, daysOfWeek []int, hour *int, timezone string, now time.Time) (*time.Time, error) {
	if timezone == "" {
		return nil, ErrInvalidAlert
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, ErrInvalidAlert
	}

	switch scheduleType {
	case AlertScheduleOnce:
		if scheduledAt == nil || !scheduledAt.After(now) {
			return nil, ErrInvalidAlert
		}
		nextRunAt := scheduledAt.UTC()
		return &nextRunAt, nil
	case AlertScheduleWeekly:
		if len(daysOfWeek) == 0 || hour == nil || *hour < 0 || *hour > 23 {
			return nil, ErrInvalidAlert
		}
		selectedDays := make(map[int]bool, len(daysOfWeek))
		for _, day := range daysOfWeek {
			if day < 0 || day > 6 {
				return nil, ErrInvalidAlert
			}
			selectedDays[day] = true
		}

		localNow := now.In(location)
		for offset := 0; offset <= 7; offset++ {
			candidateDate := localNow.AddDate(0, 0, offset)
			if !selectedDays[int(candidateDate.Weekday())] {
				continue
			}
			candidate := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), *hour, 0, 0, 0, location)
			if candidate.After(localNow) {
				nextRunAt := candidate.UTC()
				return &nextRunAt, nil
			}
		}
	}

	return nil, ErrInvalidAlert
}

func (d Device) Validate() error {
	if d.ID == "" || d.PushToken == "" {
		return ErrInvalidDevice
	}
	switch d.Platform {
	case DevicePlatformIOS, DevicePlatformAndroid:
		return nil
	default:
		return ErrInvalidDevice
	}
}

func (a *Alert) PrepareForCreate(now time.Time) error {
	a.Base = strings.ToUpper(a.Base)
	a.Target = strings.ToUpper(a.Target)
	a.Pair = BuildAlertPair(a.Base, a.Target)

	if a.ID.IsZero() {
		a.ID = primitive.NewObjectID()
	}
	if a.Status == "" {
		a.Status = AlertStatusActive
	}
	a.CreatedAt = now
	a.UpdatedAt = now

	return a.Validate()
}

func (a Alert) Validate() error {
	if a.DeviceID == "" || a.Base == "" || a.Target == "" || a.Pair == "" {
		return ErrInvalidAlert
	}
	if a.Status != AlertStatusActive && a.Status != AlertStatusTriggered {
		return ErrInvalidAlert
	}

	switch a.Type {
	case AlertTypeThreshold:
		if a.Value == 0 || (a.Direction != AlertDirectionUp && a.Direction != AlertDirectionDown) {
			return ErrInvalidAlert
		}
	case AlertTypeSchedule:
		if a.ScheduleType == AlertScheduleOnce {
			if a.ScheduledAt == nil || a.NextRunAt == nil || a.Timezone == "" {
				return ErrInvalidAlert
			}
		} else if a.ScheduleType == AlertScheduleWeekly {
			if len(a.DaysOfWeek) == 0 || a.Hour == nil || a.NextRunAt == nil || a.Timezone == "" {
				return ErrInvalidAlert
			}
			if *a.Hour < 0 || *a.Hour > 23 {
				return ErrInvalidAlert
			}
			for _, day := range a.DaysOfWeek {
				if day < 0 || day > 6 {
					return ErrInvalidAlert
				}
			}
		} else {
			return ErrInvalidAlert
		}
	default:
		return ErrInvalidAlert
	}

	return nil
}

func (r *AlertRepository) EnsureIndexes(ctx context.Context) error {
	deviceIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "push_token", Value: 1}},
		},
	}
	if _, err := r.devices().Indexes().CreateMany(ctx, deviceIndexes); err != nil {
		return err
	}

	alertIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "pair", Value: 1},
				{Key: "status", Value: 1},
				{Key: "type", Value: 1},
				{Key: "direction", Value: 1},
				{Key: "value", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "device_id", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "status", Value: 1},
				{Key: "next_run_at", Value: 1},
				{Key: "last_sent_at", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "device_id", Value: 1},
				{Key: "pair", Value: 1},
				{Key: "value", Value: 1},
				{Key: "type", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"type": AlertTypeThreshold}),
		},
	}
	_, err := r.alerts().Indexes().CreateMany(ctx, alertIndexes)
	return err
}

func (r *AlertRepository) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	var device Device
	err := r.devices().FindOne(ctx, bson.M{
		"_id":       deviceID,
		"is_active": true,
	}).Decode(&device)
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *AlertRepository) DeactivateDevice(ctx context.Context, deviceID string) error {
	now := time.Now().UTC()
	_, err := r.devices().UpdateOne(ctx, bson.M{"_id": deviceID}, bson.M{
		"$set": bson.M{
			"is_active":  false,
			"updated_at": now,
		},
	})
	return err
}

// SaveDevice upserts device state on every app launch and refreshes last_seen_at.
func (r *AlertRepository) SaveDevice(ctx context.Context, device Device) error {
	if err := device.Validate(); err != nil {
		return err
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"push_token":   device.PushToken,
			"platform":     device.Platform,
			"is_active":    true,
			"updated_at":   now,
			"last_seen_at": now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.devices().UpdateOne(ctx, bson.M{"_id": device.ID}, update, opts)
	return err
}

func (r *AlertRepository) CreateAlert(ctx context.Context, alert Alert) (primitive.ObjectID, error) {
	now := time.Now().UTC()
	if err := alert.PrepareForCreate(now); err != nil {
		return primitive.NilObjectID, err
	}

	_, err := r.alerts().InsertOne(ctx, alert)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return alert.ID, nil
}

func (r *AlertRepository) GetDeviceAlerts(ctx context.Context, deviceID string, status string) ([]Alert, error) {
	filter := bson.M{"device_id": deviceID}
	if status != "" {
		filter["status"] = status
	}

	cur, err := r.alerts().Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var alerts []Alert
	for cur.Next(ctx) {
		var alert Alert
		if err := cur.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (r *AlertRepository) GetAlert(ctx context.Context, id primitive.ObjectID, deviceID string) (*Alert, error) {
	var alert Alert
	err := r.alerts().FindOne(ctx, bson.M{
		"_id":       id,
		"device_id": deviceID,
	}).Decode(&alert)
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *AlertRepository) CountActiveAlerts(ctx context.Context, deviceID string) (int64, error) {
	return r.alerts().CountDocuments(ctx, bson.M{
		"device_id": deviceID,
		"status":    AlertStatusActive,
	})
}

func (r *AlertRepository) DeleteAlert(ctx context.Context, id primitive.ObjectID, deviceID string) error {
	result, err := r.alerts().DeleteOne(ctx, bson.M{
		"_id":       id,
		"device_id": deviceID,
	})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrAlertNotFound
	}
	return nil
}

func (r *AlertRepository) FindDueScheduleAlerts(ctx context.Context, now time.Time, limit int64) ([]Alert, error) {
	opts := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{Key: "next_run_at", Value: 1}})
	cur, err := r.alerts().Find(ctx, bson.M{
		"type":        AlertTypeSchedule,
		"status":      AlertStatusActive,
		"next_run_at": bson.M{"$lte": now},
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var alerts []Alert
	for cur.Next(ctx) {
		var alert Alert
		if err := cur.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (r *AlertRepository) ClaimScheduleAlert(ctx context.Context, alert Alert, now time.Time) (*Alert, error) {
	filter := bson.M{
		"_id":         alert.ID,
		"type":        AlertTypeSchedule,
		"status":      AlertStatusActive,
		"next_run_at": bson.M{"$lte": now},
	}

	set := bson.M{
		"updated_at": now,
	}
	if alert.ScheduleType == AlertScheduleOnce {
		set["status"] = AlertStatusTriggered
		set["triggered_at"] = now
	} else {
		nextRunAt, err := CalculateNextRunAt(alert.ScheduleType, alert.ScheduledAt, alert.DaysOfWeek, alert.Hour, alert.Timezone, now)
		if err != nil {
			return nil, err
		}
		set["next_run_at"] = nextRunAt
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var claimed Alert
	err := r.alerts().FindOneAndUpdate(ctx, filter, bson.M{"$set": set}, opts).Decode(&claimed)
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (r *AlertRepository) MarkAlertSent(ctx context.Context, id primitive.ObjectID, sentAt time.Time) error {
	_, err := r.alerts().UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"last_sent_at": sentAt,
			"updated_at":   sentAt,
		},
	})
	return err
}

func (r *AlertRepository) GetActiveThresholdAlerts(ctx context.Context, limit int64) ([]Alert, error) {
	opts := options.Find().SetLimit(limit)
	cur, err := r.alerts().Find(ctx, bson.M{
		"type":   AlertTypeThreshold,
		"status": AlertStatusActive,
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var alerts []Alert
	for cur.Next(ctx) {
		var alert Alert
		if err := cur.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (r *AlertRepository) UpdateThresholdAlert(ctx context.Context, id primitive.ObjectID, deviceID, base, target string, value float64, direction string) (*Alert, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":       id,
		"device_id": deviceID,
		"type":      AlertTypeThreshold,
	}
	update := bson.M{
		"$set": bson.M{
			"base":       strings.ToUpper(base),
			"target":     strings.ToUpper(target),
			"pair":       BuildAlertPair(base, target),
			"value":      value,
			"direction":  direction,
			"status":     AlertStatusActive,
			"updated_at": now,
		},
		"$unset": bson.M{
			"triggered_at": "",
			"trigger_rate": "",
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var alert Alert
	if err := r.alerts().FindOneAndUpdate(ctx, filter, update, opts).Decode(&alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *AlertRepository) UpdateScheduleAlert(ctx context.Context, id primitive.ObjectID, deviceID string, alert Alert) (*Alert, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":       id,
		"device_id": deviceID,
		"type":      AlertTypeSchedule,
	}

	set := bson.M{
		"schedule_type": alert.ScheduleType,
		"timezone":      alert.Timezone,
		"next_run_at":   alert.NextRunAt,
		"status":        AlertStatusActive,
		"updated_at":    now,
	}
	unset := bson.M{
		"triggered_at":  "",
		"trigger_rate":  "",
		"interval_days": "",
	}

	if alert.ScheduleType == AlertScheduleOnce {
		set["scheduled_at"] = alert.ScheduledAt
		unset["days_of_week"] = ""
		unset["hour"] = ""
	}
	if alert.ScheduleType == AlertScheduleWeekly {
		set["days_of_week"] = alert.DaysOfWeek
		set["hour"] = alert.Hour
		unset["scheduled_at"] = ""
	}

	update := bson.M{
		"$set":   set,
		"$unset": unset,
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updated Alert
	if err := r.alerts().FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *AlertRepository) FindTriggeredThresholdAlerts(ctx context.Context, base, target, direction string, currentRate float64) ([]Alert, error) {
	valueFilter := bson.M{"$lte": currentRate}
	if direction == AlertDirectionDown {
		valueFilter = bson.M{"$gte": currentRate}
	}

	filter := bson.M{
		"pair":      BuildAlertPair(base, target),
		"status":    AlertStatusActive,
		"type":      AlertTypeThreshold,
		"direction": direction,
		"value":     valueFilter,
	}

	cur, err := r.alerts().Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var alerts []Alert
	for cur.Next(ctx) {
		var alert Alert
		if err := cur.Decode(&alert); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (r *AlertRepository) MarkAlertTriggered(ctx context.Context, id primitive.ObjectID, triggerRate float64) (*Alert, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":    id,
		"type":   AlertTypeThreshold,
		"status": AlertStatusActive,
	}
	update := bson.M{
		"$set": bson.M{
			"status":       AlertStatusTriggered,
			"triggered_at": now,
			"trigger_rate": triggerRate,
			"updated_at":   now,
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var alert Alert
	if err := r.alerts().FindOneAndUpdate(ctx, filter, update, opts).Decode(&alert); err != nil {
		return nil, err
	}
	return &alert, nil
}
