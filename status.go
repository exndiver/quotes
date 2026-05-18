package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const statusStartupGrace = 10 * time.Minute

var (
	statusStartedAt = time.Now()
	statusMu        sync.RWMutex
	statusModules   = map[string]*moduleStatus{}
)

type moduleStatus struct {
	Name          string     `json:"name"`
	Enabled       bool       `json:"enabled"`
	Healthy       bool       `json:"healthy"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
	LastMessage   string     `json:"last_message,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
}

type statusPageResponse struct {
	Status    string          `json:"status"`
	Healthy   bool            `json:"healthy"`
	CheckedAt time.Time       `json:"checked_at"`
	Issues    []string        `json:"issues,omitempty"`
	Modules   []moduleStatus  `json:"modules"`
}

func statusModuleEnabled(name string) bool {
	switch name {
	case "mongodb":
		return true
	case "openexchangerates":
		return Config.DownloadRates && Config.Plugins.OpenExRates
	case "crypto":
		return Config.DownloadRates && Config.Plugins.Crypto
	case "stocks":
		return Config.DownloadRates && stocksEnabled()
	case "fuel":
		return FuelOrchestrator != nil
	case "memory_reload":
		return true
	case "history":
		return true
	default:
		return false
	}
}

func stocksEnabled() bool {
	for _, s := range Config.Stocks {
		if s.Enable {
			return true
		}
	}
	return false
}

func statusRecordAttempt(name string) {
	if !statusModuleEnabled(name) {
		return
	}
	now := time.Now()
	statusMu.Lock()
	defer statusMu.Unlock()
	m := statusGetOrCreateLocked(name)
	m.Enabled = true
	m.LastAttemptAt = &now
}

func statusRecordSuccess(name, message string) {
	if !statusModuleEnabled(name) {
		return
	}
	now := time.Now()
	statusMu.Lock()
	defer statusMu.Unlock()
	m := statusGetOrCreateLocked(name)
	m.Enabled = true
	m.LastAttemptAt = &now
	m.LastSuccessAt = &now
	m.LastMessage = message
	m.LastError = ""
	m.Healthy = true
}

func statusRecordFailure(name string, err error) {
	if !statusModuleEnabled(name) {
		return
	}
	now := time.Now()
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	statusMu.Lock()
	defer statusMu.Unlock()
	m := statusGetOrCreateLocked(name)
	m.Enabled = true
	m.LastAttemptAt = &now
	m.LastError = msg
	m.LastMessage = "update failed"
	m.Healthy = false
}

func statusRecordSkip(name, message string) {
	if !statusModuleEnabled(name) {
		return
	}
	now := time.Now()
	statusMu.Lock()
	defer statusMu.Unlock()
	m := statusGetOrCreateLocked(name)
	m.Enabled = true
	m.LastAttemptAt = &now
	m.LastMessage = message
	m.LastError = ""
	m.Healthy = true
}

func statusGetOrCreateLocked(name string) *moduleStatus {
	m, ok := statusModules[name]
	if !ok {
		m = &moduleStatus{Name: name, Enabled: statusModuleEnabled(name)}
		statusModules[name] = m
	}
	return m
}

func moduleMaxStale(name string) time.Duration {
	switch name {
	case "openexchangerates":
		day := time.Now().Weekday()
		if day == time.Sunday || day == time.Saturday {
			return 72 * time.Hour
		}
		return 26 * time.Hour
	case "crypto", "stocks", "memory_reload":
		return 15 * time.Minute
	case "history":
		return 6 * time.Hour
	case "fuel":
		if FuelOrchestrator != nil {
			if d := FuelOrchestrator.MaxStaleAfter(); d > 0 {
				return d
			}
		}
		return 48 * time.Hour
	case "mongodb":
		return 0
	default:
		return 24 * time.Hour
	}
}

func evaluateStatus() (healthy bool, issues []string, modules []moduleStatus) {
	healthy = true
	issues = []string{}

	mongoOK := client.Ping(context.TODO(), nil) == nil
	mongoMod := moduleStatus{
		Name:    "mongodb",
		Enabled: true,
		Healthy: mongoOK,
	}
	if mongoOK {
		mongoMod.LastMessage = "ping ok"
	} else {
		mongoMod.LastError = "database unreachable"
		mongoMod.LastMessage = "ping failed"
		issues = append(issues, "mongodb: database unreachable")
		healthy = false
	}
	modules = append(modules, mongoMod)

	names := []string{"openexchangerates", "crypto", "stocks", "fuel", "memory_reload", "history"}
	inGrace := time.Since(statusStartedAt) < statusStartupGrace

	for _, name := range names {
		if !statusModuleEnabled(name) {
			continue
		}

		statusMu.RLock()
		m, ok := statusModules[name]
		if !ok {
			statusMu.RUnlock()
			if !inGrace {
				issues = append(issues, name+": no successful update recorded yet")
			}
			modules = append(modules, moduleStatus{
				Name:    name,
				Enabled: true,
				Healthy: inGrace,
			})
			continue
		}
		snap := *m
		snap.Name = name
		snap.Enabled = true
		statusMu.RUnlock()

		modHealthy := true
		if snap.LastError != "" {
			if snap.LastSuccessAt == nil || (snap.LastAttemptAt != nil && snap.LastAttemptAt.After(*snap.LastSuccessAt)) {
				modHealthy = false
				issues = append(issues, fmt.Sprintf("%s: %s", name, snap.LastError))
			}
		}

		maxStale := moduleMaxStale(name)
		if maxStale > 0 && snap.LastSuccessAt != nil {
			if time.Since(*snap.LastSuccessAt) > maxStale {
				modHealthy = false
				issues = append(issues, fmt.Sprintf("%s: last success too old (%s ago)", name, formatDuration(time.Since(*snap.LastSuccessAt))))
			}
		} else if maxStale > 0 && snap.LastSuccessAt == nil && !inGrace {
			modHealthy = false
			issues = append(issues, name+": never updated successfully")
		}

		snap.Healthy = modHealthy
		modules = append(modules, snap)
		if !modHealthy {
			healthy = false
		}
	}

	return healthy, issues, modules
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func buildStatusResponse() (code int, body []byte, contentType string) {
	healthy, issues, modules := evaluateStatus()
	resp := statusPageResponse{
		CheckedAt: time.Now().UTC(),
		Modules:   modules,
		Issues:    issues,
		Healthy:   healthy,
	}
	if healthy {
		resp.Status = "ok"
		code = http.StatusOK
	} else {
		resp.Status = "error"
		code = http.StatusServiceUnavailable
	}
	body, _ = json.MarshalIndent(resp, "", "  ")
	return code, body, "application/json; charset=utf-8"
}

func statusHTTP(w http.ResponseWriter, r *http.Request) {
	code, body, contentType := buildStatusResponse()

	w.Header().Set("Cache-Control", "no-store")

	if r.Method == http.MethodHead {
		w.WriteHeader(code)
		return
	}

	if strings.Contains(r.Header.Get("Accept"), "text/html") || r.URL.Query().Get("format") == "html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		_, issues, modules := evaluateStatus()
		fmt.Fprintf(w, "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Quotes status</title></head><body>")
		if code == http.StatusOK {
			fmt.Fprintf(w, "<h1>OK</h1>")
		} else {
			fmt.Fprintf(w, "<h1>ERROR</h1>")
		}
		if len(issues) > 0 {
			fmt.Fprintf(w, "<h2>Issues</h2><ul>")
			for _, i := range issues {
				fmt.Fprintf(w, "<li>%s</li>", htmlEscape(i))
			}
			fmt.Fprintf(w, "</ul>")
		}
		fmt.Fprintf(w, "<h2>Modules</h2><table border=\"1\" cellpadding=\"6\"><tr><th>Module</th><th>OK</th><th>Last success</th><th>Last attempt</th><th>Message</th><th>Error</th></tr>")
		for _, m := range modules {
			ok := "yes"
			if !m.Healthy {
				ok = "no"
			}
			fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				htmlEscape(m.Name), ok, formatTimePtr(m.LastSuccessAt), formatTimePtr(m.LastAttemptAt),
				htmlEscape(m.LastMessage), htmlEscape(m.LastError))
		}
		fmt.Fprintf(w, "</table></body></html>")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.UTC().Format(time.RFC3339)
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
