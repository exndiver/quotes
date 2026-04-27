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
  "rate_min_step": 0.01
}
```

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

Schedule alerts use the same rate calculation. For one-time schedules, `scheduled_at` must be in the future. For recurring schedules, `interval_days` must be greater than `0`.

### List Alerts

`GET /alerts?device_id=uuid-123`

Optional filter: `status=active`.

### Delete Alert

`DELETE /alerts/{id}?device_id=uuid-123`

The alert is deleted only when it belongs to the provided `device_id`.

All alert endpoints also have `/api/...` aliases for the existing router style.
