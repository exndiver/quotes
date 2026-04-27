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
	IntervalDays int        `json:"interval_days"`
}

type createAlertResponse struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	Direction         string  `json:"direction,omitempty"`
	CurrentRate       float64 `json:"current_rate"`
	ActiveAlertsCount int64   `json:"active_alerts_count"`
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
		IntervalDays: req.IntervalDays,
	}

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
	}

	if req.Type == AlertTypeSchedule && req.ScheduleType == AlertScheduleOnce {
		if req.ScheduledAt == nil || !req.ScheduledAt.After(time.Now().UTC()) {
			code = http.StatusBadRequest
			resp := writeAlertAPIError(w, code, "INVALID_SCHEDULED_AT")
			return code, mn, 4, string(resp), rbody
		}
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
		CurrentRate:       currentRate,
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

	resp := writeAlertJSON(w, code, alerts)
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
