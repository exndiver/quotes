# quotes

1. go get -u github.com/gorilla/mux
2. go get -u github.com/gorilla/handlers
3. go get -u go.mongodb.org/mongo-driver
4. go get -u github.com/antchfx/htmlquery
5. go get -u github.com/exndiver/cache/memory
6. go get -u github.com/exndiver/feedback

## Alerts API

Server is the only source of truth for rates. Clients never send `current_rate`; it is calculated on the server from rates stored in MongoDB and reused by alert creation and trigger checks.

Config defaults:

```json
{
  "alerts_active_limit": 100,
  "rate_min_step": 0.01,
  "alerts_workers": {
    "enabled": true,
    "schedule_interval": "5m",
    "threshold_interval": "30s",
    "schedule_batch_size": 50,
    "threshold_batch_size": 100
  },
  "firebase": {
    "credentials_file": ""
  }
}
```

Set `firebase.credentials_file` to a mounted Firebase service account JSON file, for example `config/firebase-service-account.json`, to start alert workers.

`current_rate` is calculated as `target_rate / base_rate` for currency category `0`. For the same `base` and `target`, the rate is `1`.

### Register Device

`POST /device/register`

```json
{
  "device_id": "uuid-123",
  "push_token": "token",
  "platform": "ios"
}
```

The server upserts the device and refreshes `push_token`, `updated_at`, and `last_seen_at`.

### Create Alert

`POST /alerts`

Threshold request:

```json
{
  "device_id": "uuid-123",
  "type": "threshold",
  "base": "EUR",
  "target": "USD",
  "value": 1.1
}
```

The server checks `alerts_active_limit`, calculates `current_rate`, rejects values closer than `rate_min_step`, sets `direction` automatically, builds `pair`, and saves the alert as `active`.

Response:

```json
{
  "id": "alert_id",
  "status": "active",
  "direction": "up",
  "current_rate": 1.09,
  "active_alerts_count": 7
}
```

Schedule alerts use the same rate calculation. For one-time schedules, `scheduled_at` must be in the future. For weekly schedules, `days_of_week`, `hour`, and `timezone` are required.

One-time schedule request:

```json
{
  "device_id": "uuid-123",
  "type": "schedule",
  "schedule_type": "once",
  "base": "EUR",
  "target": "USD",
  "scheduled_at": "2026-04-28T15:00:00Z",
  "timezone": "Europe/Brussels"
}
```

Weekly schedule request:

```json
{
  "device_id": "uuid-123",
  "type": "schedule",
  "schedule_type": "weekly",
  "base": "EUR",
  "target": "USD",
  "days_of_week": [1, 3, 5],
  "hour": 9,
  "timezone": "Europe/Brussels"
}
```

Only `once` and `weekly` schedules are supported. The server validates `timezone`, calculates `next_run_at`, and stores schedule alerts as `active`.

### List Alerts

`GET /alerts?device_id=uuid-123`

Optional filter: `status=active`.

### Update Alert

`PUT /alerts/{id}`

```json
{
  "device_id": "uuid-123",
  "base": "USD",
  "target": "EUR",
  "value": 0.86
}
```

The server updates an existing threshold alert only when it belongs to the provided `device_id`. It recalculates `current_rate`, `direction`, and `pair`, then sets the alert back to `active`. Unlike creation, update accepts the provided value as-is so users can edit an existing alert to the current UI step.

Schedule update examples:

```json
{
  "device_id": "uuid-123",
  "schedule_type": "once",
  "scheduled_at": "2026-04-28T15:00:00Z",
  "timezone": "Europe/Brussels"
}
```

```json
{
  "device_id": "uuid-123",
  "schedule_type": "weekly",
  "days_of_week": [2, 4],
  "hour": 18,
  "timezone": "Europe/Brussels"
}
```

### Delete Alert

`DELETE /alerts/{id}?device_id=uuid-123`

The alert is deleted only when it belongs to the provided `device_id`.

All alert endpoints also have `/api/...` aliases for the existing router style.

### Alert Workers

The service starts two independent workers when `alerts_workers.enabled` is `true` and Firebase credentials are configured.

Schedule worker:

```json
{
  "type": "schedule",
  "status": "active",
  "next_run_at": { "$lte": "now" }
}
```

Each due schedule alert is claimed with an atomic `findOneAndUpdate`. `once` alerts are marked `triggered`; `weekly` alerts get a recalculated `next_run_at` using their `timezone`.

Threshold worker:

```json
{
  "type": "threshold",
  "status": "active"
}
```

The worker recalculates the current rate server-side and triggers only when `up` means `current >= value` or `down` means `current <= value`. Triggering is atomic on `{ _id, type: "threshold", status: "active" }`.

Pushes are sent through Firebase Cloud Messaging. If Firebase reports an unregistered or invalid token, the device is marked inactive.
