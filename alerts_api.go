package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type apiErrorResponse struct {
	Error string `json:"error"`
}

type createAlertRequest struct {
	DeviceID     string     `json:"device_id"`
	Type         string     `json:"type"`
	Base         string     `json:"base"`
	Target       string     `json:"target"`
	Value        float64    `json:"value"`
	ScheduleType string     `json:"schedule_type"`
	ScheduledAt  *time.Time `json:"scheduled_at"`
	DaysOfWeek   []int      `json:"days_of_week"`
	Hour         *int       `json:"hour"`
	Timezone     string     `json:"timezone"`
}

type createAlertResponse struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	Direction         string     `json:"direction,omitempty"`
	CurrentRate       *float64   `json:"current_rate,omitempty"`
	NextRunAt         *time.Time `json:"next_run_at,omitempty"`
	ActiveAlertsCount int64      `json:"active_alerts_count"`
}

type updateAlertRequest struct {
	DeviceID     string     `json:"device_id"`
	Base         string     `json:"base"`
	Target       string     `json:"target"`
	Value        float64    `json:"value"`
	ScheduleType string     `json:"schedule_type"`
	ScheduledAt  *time.Time `json:"scheduled_at"`
	DaysOfWeek   []int      `json:"days_of_week"`
	Hour         *int       `json:"hour"`
	Timezone     string     `json:"timezone"`
}

type updateAlertResponse struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	Direction   string     `json:"direction,omitempty"`
	CurrentRate *float64   `json:"current_rate,omitempty"`
	NextRunAt   *time.Time `json:"next_run_at,omitempty"`
}

type alertAPIResponse struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Base         string     `json:"base"`
	Target       string     `json:"target"`
	Pair         string     `json:"pair,omitempty"`
	Status       string     `json:"status"`
	Value        float64    `json:"value,omitempty"`
	Direction    string     `json:"direction,omitempty"`
	ScheduleType string     `json:"schedule_type,omitempty"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty"`
	DaysOfWeek   []int      `json:"days_of_week,omitempty"`
	Hour         *int       `json:"hour,omitempty"`
	Timezone     string     `json:"timezone,omitempty"`
	NextRunAt    *time.Time `json:"next_run_at,omitempty"`
	TriggeredAt  *time.Time `json:"triggered_at,omitempty"`
	TriggerRate  float64    `json:"trigger_rate,omitempty"`
}

type deleteAlertRequest struct {
	DeviceID string `json:"device_id"`
}

func writeAlertJSON(w http.ResponseWriter, code int, value interface{}) []byte {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp, _ := json.Marshal(value)
	w.Write(resp)
	return resp
}

func writeAlertAPIError(w http.ResponseWriter, code int, errCode string) []byte {
	return writeAlertJSON(w, code, apiErrorResponse{Error: errCode})
}

func toAlertAPIResponses(alerts []Alert) []alertAPIResponse {
	responses := make([]alertAPIResponse, 0, len(alerts))
	for _, alert := range alerts {
		responses = append(responses, alertAPIResponse{
			ID:           alert.ID.Hex(),
			Type:         alert.Type,
			Base:         alert.Base,
			Target:       alert.Target,
			Pair:         alert.Pair,
			Status:       alert.Status,
			Value:        alert.Value,
			Direction:    alert.Direction,
			ScheduleType: alert.ScheduleType,
			ScheduledAt:  alert.ScheduledAt,
			DaysOfWeek:   alert.DaysOfWeek,
			Hour:         alert.Hour,
			Timezone:     alert.Timezone,
			NextRunAt:    alert.NextRunAt,
			TriggeredAt:  alert.TriggeredAt,
			TriggerRate:  alert.TriggerRate,
		})
	}
	return responses
}

func registerDeviceAPI(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "RegisterDevice"
	level := 6
	code := http.StatusOK

	var device Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "BAD_REQUEST")
		return code, mn, 4, string(resp), ""
	}
	rbodyB, _ := json.Marshal(device)
	rbody := string(rbodyB)

	if err := NewAlertRepository(client).SaveDevice(context.Background(), device); err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "INVALID_DEVICE")
		return code, mn, 4, string(resp), rbody
	}

	resp := writeAlertJSON(w, code, map[string]string{"status": "ok"})
	return code, mn, level, string(resp), rbody
}

func createAlertAPI(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "CreateAlert"
	level := 6
	code := http.StatusOK

	var req createAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "BAD_REQUEST")
		return code, mn, 4, string(resp), ""
	}
	req.Base = strings.ToUpper(req.Base)
	req.Target = strings.ToUpper(req.Target)
	rbodyB, _ := json.Marshal(req)
	rbody := string(rbodyB)

	if req.DeviceID == "" || req.Type == "" || req.Base == "" || req.Target == "" {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "INVALID_ALERT")
		return code, mn, 4, string(resp), rbody
	}

	repo := NewAlertRepository(client)
	activeCount, err := repo.CountActiveAlerts(context.Background(), req.DeviceID)
	if err != nil {
		code = http.StatusInternalServerError
		resp := writeAlertAPIError(w, code, "DB_ERROR")
		return code, mn, 3, string(resp), rbody
	}
	if activeCount >= int64(Config.AlertsActiveLimit) {
		code = http.StatusForbidden
		resp := writeAlertAPIError(w, code, "LIMIT_REACHED")
		return code, mn, 4, string(resp), rbody
	}

	currentRate, err := CalculateCurrentRate(req.Base, req.Target)
	if err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "RATE_NOT_FOUND")
		return code, mn, 4, string(resp), rbody
	}

	alert := Alert{
		DeviceID:     req.DeviceID,
		Type:         req.Type,
		Base:         req.Base,
		Target:       req.Target,
		Value:        req.Value,
		ScheduleType: req.ScheduleType,
		ScheduledAt:  req.ScheduledAt,
		DaysOfWeek:   req.DaysOfWeek,
		Hour:         req.Hour,
		Timezone:     req.Timezone,
	}
	var responseCurrentRate *float64
	var responseNextRunAt *time.Time

	if req.Type == AlertTypeThreshold {
		if req.Value <= 0 {
			code = http.StatusBadRequest
			resp := writeAlertAPIError(w, code, "INVALID_ALERT")
			return code, mn, 4, string(resp), rbody
		}
		if math.Abs(req.Value-currentRate) < Config.RateMinStep {
			code = http.StatusBadRequest
			resp := writeAlertAPIError(w, code, "VALUE_TOO_CLOSE_TO_CURRENT")
			return code, mn, 4, string(resp), rbody
		}
		if req.Value > currentRate {
			alert.Direction = AlertDirectionUp
		} else {
			alert.Direction = AlertDirectionDown
		}
		responseCurrentRate = &currentRate
	}

	if req.Type == AlertTypeSchedule {
		nextRunAt, err := CalculateNextRunAt(req.ScheduleType, req.ScheduledAt, req.DaysOfWeek, req.Hour, req.Timezone, time.Now().UTC())
		if err != nil {
			code = http.StatusBadRequest
			resp := writeAlertAPIError(w, code, "INVALID_SCHEDULE")
			return code, mn, 4, string(resp), rbody
		}
		alert.NextRunAt = nextRunAt
		responseNextRunAt = nextRunAt
	}

	id, err := repo.CreateAlert(context.Background(), alert)
	if err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "INVALID_ALERT")
		return code, mn, 4, string(resp), rbody
	}

	resp := writeAlertJSON(w, code, createAlertResponse{
		ID:                id.Hex(),
		Status:            AlertStatusActive,
		Direction:         alert.Direction,
		CurrentRate:       responseCurrentRate,
		NextRunAt:         responseNextRunAt,
		ActiveAlertsCount: activeCount + 1,
	})
	return code, mn, level, string(resp), rbody
}

func getAlertsAPI(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "GetAlerts"
	level := 6
	code := http.StatusOK
	rbody := ""

	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "DEVICE_ID_REQUIRED")
		return code, mn, 4, string(resp), rbody
	}

	alerts, err := NewAlertRepository(client).GetDeviceAlerts(context.Background(), deviceID, r.URL.Query().Get("status"))
	if err != nil {
		code = http.StatusInternalServerError
		resp := writeAlertAPIError(w, code, "DB_ERROR")
		return code, mn, 3, string(resp), rbody
	}

	resp := writeAlertJSON(w, code, toAlertAPIResponses(alerts))
	return code, mn, level, string(resp), rbody
}

func updateAlertAPI(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "UpdateAlert"
	level := 6
	code := http.StatusOK

	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "INVALID_ALERT_ID")
		return code, mn, 4, string(resp), ""
	}

	var req updateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "BAD_REQUEST")
		return code, mn, 4, string(resp), ""
	}
	req.Base = strings.ToUpper(req.Base)
	req.Target = strings.ToUpper(req.Target)
	rbodyB, _ := json.Marshal(req)
	rbody := string(rbodyB)

	if req.DeviceID == "" {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "INVALID_ALERT")
		return code, mn, 4, string(resp), rbody
	}

	if req.ScheduleType != "" {
		nextRunAt, err := CalculateNextRunAt(req.ScheduleType, req.ScheduledAt, req.DaysOfWeek, req.Hour, req.Timezone, time.Now().UTC())
		if err != nil {
			code = http.StatusBadRequest
			resp := writeAlertAPIError(w, code, "INVALID_SCHEDULE")
			return code, mn, 4, string(resp), rbody
		}
		alert, err := NewAlertRepository(client).UpdateScheduleAlert(context.Background(), id, req.DeviceID, Alert{
			ScheduleType: req.ScheduleType,
			ScheduledAt:  req.ScheduledAt,
			DaysOfWeek:   req.DaysOfWeek,
			Hour:         req.Hour,
			Timezone:     req.Timezone,
			NextRunAt:    nextRunAt,
		})
		if err != nil {
			code = http.StatusNotFound
			resp := writeAlertAPIError(w, code, "ALERT_NOT_FOUND")
			return code, mn, 4, string(resp), rbody
		}

		resp := writeAlertJSON(w, code, updateAlertResponse{
			ID:        alert.ID.Hex(),
			Status:    alert.Status,
			NextRunAt: alert.NextRunAt,
		})
		return code, mn, level, string(resp), rbody
	}

	if req.Base == "" || req.Target == "" || req.Value <= 0 {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "INVALID_ALERT")
		return code, mn, 4, string(resp), rbody
	}

	currentRate, err := CalculateCurrentRate(req.Base, req.Target)
	if err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "RATE_NOT_FOUND")
		return code, mn, 4, string(resp), rbody
	}

	direction := AlertDirectionDown
	if req.Value > currentRate {
		direction = AlertDirectionUp
	}

	alert, err := NewAlertRepository(client).UpdateThresholdAlert(context.Background(), id, req.DeviceID, req.Base, req.Target, req.Value, direction)
	if err != nil {
		code = http.StatusNotFound
		resp := writeAlertAPIError(w, code, "ALERT_NOT_FOUND")
		return code, mn, 4, string(resp), rbody
	}

	resp := writeAlertJSON(w, code, updateAlertResponse{
		ID:          alert.ID.Hex(),
		Status:      alert.Status,
		Direction:   alert.Direction,
		CurrentRate: &currentRate,
	})
	return code, mn, level, string(resp), rbody
}

func deleteAlertAPI(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "DeleteAlert"
	level := 6
	code := http.StatusOK

	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "INVALID_ALERT_ID")
		return code, mn, 4, string(resp), ""
	}

	deviceID := r.URL.Query().Get("device_id")
	var req deleteAlertRequest
	if deviceID == "" {
		_ = json.NewDecoder(r.Body).Decode(&req)
		deviceID = req.DeviceID
	}
	rbodyB, _ := json.Marshal(req)
	rbody := string(rbodyB)

	if deviceID == "" {
		code = http.StatusBadRequest
		resp := writeAlertAPIError(w, code, "DEVICE_ID_REQUIRED")
		return code, mn, 4, string(resp), rbody
	}

	if err := NewAlertRepository(client).DeleteAlert(context.Background(), id, deviceID); err != nil {
		code = http.StatusNotFound
		resp := writeAlertAPIError(w, code, "ALERT_NOT_FOUND")
		return code, mn, 4, string(resp), rbody
	}

	resp := writeAlertJSON(w, code, map[string]string{"status": "deleted"})
	return code, mn, level, string(resp), rbody
}
