# ROEST Coffee Roaster API

**Base URL:** `https://api.roestcoffee.com/`
**Auth:** `Authorization: Bearer <token>`
**Framework:** Django REST Framework

## Authentication

Tokens are obtained via OAuth2 client credentials grant:

```bash
curl 'https://api.roestcoffee.com/o/token/' \
  -H 'Content-Type: application/json' \
  -d '{
    "grant_type": "client_credentials",
    "client_id": "your-client-id",
    "client_secret": "your-client-secret"
  }'
```

The `client_id` can be found via the `/applications/` endpoint. Use the returned access token as `Authorization: Bearer <token>` on all subsequent requests.

## Pagination

All list endpoints return paginated responses:

```json
{
  "count": 814,
  "next": "https://api.roestcoffee.com/logs/?page=2",
  "previous": null,
  "current_page_no": 1,
  "final_page_no": 33,
  "page_list": [
    { "url": "https://api.roestcoffee.com/logs/", "number": 1 },
    { "url": "https://api.roestcoffee.com/logs/?page=2", "number": 2 }
  ],
  "results": [...]
}
```

Default page size is 25 results.

---

## Endpoints

### GET `/users/`

Returns the authenticated user.

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | User ID |
| `url` | string | Resource URL |
| `username` | string | Username |
| `first_name` | string | First name |
| `email` | string | Email address |
| `is_staff` | bool | Staff status |
| `customers` | array | List of associated customers (id, name, url, enabled_beta_features) |
| `user_setting` | object | User preferences (see below) |
| `two_factor` | bool | Whether 2FA is enabled |

**`user_setting` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `temperature_metric` | int | 0 = Celsius, 1 = Fahrenheit |
| `weight_metric` | int | 0 = grams, 1 = ounces |
| `color_theme` | int | UI color theme |
| `ror_interval` | int | Rate of Rise interval (S100/S200) |
| `ror_interval_p3000` | int | Rate of Rise interval (P3000) |
| `max_chart_temperature_s100` | int | Max chart temp for S100/S200 |
| `max_chart_temperature_p3000` | int | Max chart temp for P3000 |
| `grouped_series_tooltip` | bool | Group series in tooltip |
| `show_individual_sensors` | bool | Show individual sensor lines |
| `list_settings` | array | Column visibility per list view |

---

### GET `/customers/`

Returns organizations the user belongs to.

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `name` | string | Organization name |
| `enabled_beta_features` | array | Beta feature flags |

---

### GET/POST `/machines/`

List or create roasting machines.

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `id` | int | Machine ID |
| `name` | string | Machine name |
| `slug` | string | URL-friendly name |
| `is_p2000` | bool | Whether this is a P2000/P3000 model |
| `particle_id` | string? | Particle IoT device ID |
| `machine_id` | string | Unique machine identifier |
| `machine_image` | string | Machine image key (e.g. `s200_woodgrain`) |
| `notes` | string? | User notes |
| `mqtt_config` | object | MQTT connection details (see below) |
| `sensor_config` | object | Sensor configuration (see below) |
| `bbp_profile` | object | Between-batch profile (inline profile object) |

**`mqtt_config` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `subscribe_password` | string | MQTT subscribe password |
| `topic` | string | MQTT topic (e.g. `roest/p2000/<machine_id>/#`) |
| `username` | string | MQTT username (e.g. `sub_<machine_id>`) |

**`sensor_config` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `has_inlet` | bool | Has inlet temperature sensor |
| `has_drum` | bool | Has drum temperature sensor |
| `has_pressure` | bool | Has pressure sensor |
| `has_reversible_drum` | bool | Drum direction is reversible |
| `rtd_count` | int | Number of RTD sensors |
| `tc_count` | int | Number of thermocouple sensors |
| `enabled_sensors` | array | List of active sensors (name, sensor_type, order, color, label) |

**Sensor types:**

| Value | Meaning |
|-------|---------|
| 0 | Inlet |
| 2 | Drum |
| 3 | Exhaust |

---

### GET `/machineslots/?machine=<machine_id>`

Profile slots loaded on a machine. Each machine has up to 5 slots.

**Required filter:** `machine` (machine ID)

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `slot_index` | int | Slot number (0-4) |
| `machine` | string | Machine URL |
| `loaded_profile` | string | Currently loaded profile URL |
| `requested_profile` | string | Requested profile URL |
| `loaded_timestamp` | datetime | When the profile was loaded |
| `requested_profile_data` | object | Full inline profile object |

---

### GET `/profiles/?customer=<id>` or `/profiles/?share_uuid=<uuid>`

Roast profiles defining temperature, fan, RPM, and power curves.

**Required filter:** `customer` or `share_uuid`

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `id` | int | Profile ID |
| `name` | string | Profile name |
| `profile_type` | int | Profile type (see below) |
| `parent` | string? | Parent profile URL (for versioning) |
| `is_leaf` | bool | Whether this is the latest version |
| `customer` | string | Owner customer URL |
| `machinetype` | int | Target machine type |
| `created` | datetime | Creation timestamp |
| `modified` | datetime | Last modified timestamp |
| `temperature_bezier` | array | Temperature curve as bezier control points `[[temp, value], ...]` |
| `power_bezier` | array | Heater power curve (bezier) |
| `rpm_bezier` | array | Drum RPM curve (bezier) |
| `fan_bezier` | array | Fan speed curve (bezier) |
| `preheat_temperature` | float | Preheat temperature |
| `batch_weight` | float | Default batch weight (grams) |
| `runlength` | int | Profile run length |
| `end_condition` | int | End condition type (1 = time, 2 = other) |
| `end_condition_value` | float | End condition value (msec for time-based) |
| `is_bbp_profile` | bool | Between-batch profile flag |
| `reversed_drum_direction` | bool | Reverse drum rotation |
| `share_uuid` | string? | Private share UUID |
| `public_share_uuid` | string? | Public share UUID |
| `public_share_disabled` | bool | Public sharing disabled |
| `downloads` | int | Download count |
| `author_name` | string | Author/customer name |
| `selected_filters` | array | Selected filters |
| `metadata_image` | string? | Profile image URL |
| `ratings` | array | User ratings |
| `notes` | string? | Profile notes |
| `is_verified` | bool | Verified by ROEST |

**Profile types:**

| Value | Meaning |
|-------|---------|
| 0 | Standard / Warmup |
| 7 | Custom |

---

### GET/POST `/logs/`

Roast log records. Each log represents a single roast session.

**Required filter:** `customer`

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `id` | int | Log ID |
| `batch_no` | int | Sequential batch number |
| `batch_uuid` | int | Unique batch identifier |
| `bean_name` | string | Name of the bean roasted |
| `name` | string? | Optional log name |
| `machine` | string | Machine URL |
| `machine_slug` | string | Machine slug |
| `machine_prod_type` | int | Machine product type |
| `profile` | string | Profile URL used |
| `profile_id` | int | Profile ID |
| `profile_name` | string? | Profile name override |
| `profile_data` | object | Inline profile summary (id, name, profile_type, batch_weight, reversed_drum_direction) |
| `sensor_config` | object | Sensor config at time of roast |
| `start_timestamp` | datetime | Roast start time |
| `end_timestamp` | datetime | Roast end time |
| `firstcrack_timestamp` | datetime | First crack time |
| `drop_timestamp` | datetime | Bean drop time |
| `start_weight` | float | Green bean weight (grams) |
| `end_weight` | float | Roasted bean weight (grams) |
| `fc_temp` | float | Bean temperature at first crack |
| `end_temp` | float | Bean temperature at end |
| `charge_drum_temp` | float | Drum temperature at charge |
| `end_drum_temp` | float | Drum temperature at end |
| `dryend_event_msec` | int | Dry end event time (msec from start) |
| `firstcrack_event_msec` | int | First crack event time (msec from start) |
| `events` | array | List of events (see below) |
| `event_flags` | int | Bitmask of which events occurred |
| `whole_bean_color` | float? | Whole bean color reading |
| `ground_bean_color` | float? | Ground bean color reading |
| `green_bean_color` | float? | Green bean color reading |
| `inventory` | string? | Inventory URL |
| `inventory_id` | int? | Inventory ID |
| `latest_cupping_score` | float? | Latest cupping score |
| `first_comment` | string? | First comment text |
| `share_uuid` | string? | Share UUID |
| `version` | int | Log version |

**Event types:**

| Value | Meaning |
|-------|---------|
| 0 | Charge (beans loaded) |
| 1 | Drop (beans dropped) |
| 4 | Dry end |
| 5 | First crack |

**Derived fields:**

- **Weight loss %:** `(start_weight - end_weight) / start_weight * 100`
- **Roast duration:** `end_timestamp - start_timestamp`
- **Development time:** `firstcrack_event_msec` to drop event msec
- **Development ratio:** development time / total roast time

---

### GET `/datapoints/?log=<log_id>`

Time-series sensor data for a roast log. Sampled at ~1 second intervals.

**Required filter:** `log` (log ID) or `share_uuid`

**Tip:** pass `page_size=all` to get every datapoint in a single response (bypasses the 25/page default), e.g. `/datapoints/?page_size=all&log=3966514`. This is what the web app uses to seed a roast chart on page load before switching to the live MQTT feed.

**Gotcha:** with `page_size=all` the endpoint returns a **bare JSON array** of datapoints, not the usual paginated envelope (`{count, results, …}`). Empty results also come back as `[]`. Parse defensively for both shapes.

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `msec` | int | Milliseconds from roast start |
| `data_type` | int | Data type (1 = normal reading) |
| `bt` | float | Bean temperature (primary RTD) |
| `et` | float | Environment temperature |
| `target` | float | Target temperature from profile |
| `fan` | int | Fan speed (%) |
| `heat` | int | Heater power (%) |
| `rpm` | int | Drum RPM |
| `drum_temp` | float | Drum thermocouple reading |
| `inlet_temp` | float | Inlet thermocouple reading |
| `air_pressure` | float | Air pressure reading |
| `crack` | int | Crack detection signal |
| `ror` | int? | Rate of Rise (integer) |
| `ror_float` | float? | Rate of Rise (precise) |
| `bt2` | float? | Secondary bean temp |
| `et2` | float? | Secondary environment temp |
| `pcb_temperature` | float | PCB board temperature |
| `pcb_humidity` | float | PCB humidity reading |
| `precise_data` | bool | Whether data is high-precision |
| `timestamp` | datetime? | Absolute timestamp |
| `event_type` | int? | Event type if this is an event datapoint |
| `sec` | int? | Seconds (alternative to msec) |
| `rtd0` - `rtd7` | float? | RTD sensor readings (up to 8) |
| `tc0` - `tc12` | float? | Thermocouple readings (up to 13) |
| `has_valid_raw_sensors` | bool | Whether raw sensor data is valid |
| `log` | string | Parent log URL |

---

### GET `/inventories/?customer=<id>`

Green bean inventory management.

**Required filter:** `customer`

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `id` | int | Inventory ID |
| `customer_id` | int | Customer ID |
| `customer` | string | Customer URL |
| `name` | string | Bean name |
| `cultivar` | string | Bean cultivar/variety |
| `farm` | string | Farm name |
| `region` | string | Growing region |
| `country` | string | Country of origin |
| `exporter` | string | Exporter name |
| `importer` | string | Importer name |
| `producer` | string | Producer name |
| `bean_size` | float? | Bean screen size |
| `drying_speed` | float? | Drying speed |
| `bean_process` | int? | Processing method |
| `moisture` | float | Moisture content (%) |
| `water_activity` | float | Water activity |
| `elevation` | float | Growing elevation (masl) |
| `density` | float | Bean density |
| `green_bean_color` | float | Green bean color reading |
| `quality` | int | Quality grade |
| `reg_date` | datetime | Registration date |
| `initial_weight` | float | Initial weight (grams) |
| `current_weight` | float | Current remaining weight (grams) |
| `price` | float | Price |
| `notes` | string | Notes |
| `is_archived` | bool | Archived status |
| `created` | datetime | Creation timestamp |
| `modified` | datetime | Last modified timestamp |

---

### GET `/cuppingforms/`

Cupping evaluation form templates. No filter required.

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Form ID |
| `url` | string | Resource URL |
| `name` | string | Form name (e.g. "CoE form v1") |
| `status` | int | Form status (2 = active) |
| `enable_descriptors` | bool | Enable flavor descriptors |
| `enable_notes` | bool | Enable notes field |
| `score_calculation` | int | Score calculation method |
| `score_constant` | float | Constant added to score (e.g. 36.0 for CoE) |
| `form_fields` | array | List of form fields (see below) |

**`form_fields` entry:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Field ID |
| `field_label` | string | Display label (e.g. "Clean cup", "Roast level") |
| `field_type_name` | string | Field type key |
| `field_type` | int | Field type ID |
| `field_values` | array | Possible values (id, label, value, parent_field_type) |
| `default_value` | float | Default value |
| `included_in_score` | bool | Whether field contributes to total score |
| `order` | int? | Display order |

---

### GET `/cuppingsessions/?customer=<id>`

Cupping session records.

**Required filter:** `customer`

*Fields: not yet observed (no data available)*

---

### GET `/comments/?log=<log_id>`

Comments attached to roast logs.

**Required filter:** `log` (log ID)

*Fields: not yet observed (no data available)*

---

### GET `/flatpages/`

Static content pages (changelog, help text, etc.). No filter required.

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Page path (e.g. `/changelog/`) |
| `title` | string | Page title |
| `content` | string | HTML content |

---

### GET `/systemmessages/`

System-wide notifications. No filter required.

*Fields: not yet observed (no data available)*

---

### GET `/applications/`

OAuth2 applications registered by the user.

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | Resource URL |
| `client_id` | string | OAuth2 client ID |
| `name` | string | Application name |

---

## Entity Relationships

```
Customer
  ├── Machine
  │     ├── MachineSlot (5 per machine)
  │     │     └── Profile (loaded/requested)
  │     └── mqtt_config (live data streaming)
  ├── Profile (tree structure via parent field)
  │     └── Bezier curves (temperature, fan, rpm, power)
  ├── Log (roast session record)
  │     ├── Datapoint (time-series sensor data, ~1/sec)
  │     ├── Comment
  │     └── Event (charge, dry end, first crack, drop)
  ├── Inventory (green bean stock)
  ├── CuppingSession
  │     └── CuppingForm (evaluation template)
  └── Application (OAuth2 apps)
```

## MQTT Live Data

Machines expose MQTT credentials for real-time sensor streaming:

- **Broker (WebSocket):** `wss://client.roestcoffee.com:8083/mqtt` — MQTT over secure WebSocket (EMQX default WSS port), confirmed from the web app.
- **Topic pattern:** `roest/p2000/<machine_id>/#`
- **Auth:** username/password from `machine.mqtt_config`

**Live-monitoring pattern (as used by the web app):**

1. `GET /datapoints/?page_size=all&log=<log_id>` — seed with all datapoints so far for the ongoing roast.
2. Connect to `wss://client.roestcoffee.com:8083/mqtt` and subscribe to `roest/p2000/<machine_id>/#`.
3. Append streamed datapoints to the seeded series as they arrive.

**MQTT payload format (captured live):**

Messages publish to `roest/p2000/<machine_id>/v2` at ~1/sec. Payload is JSON:

```json
{
  "batch_uuid": "3104759875",
  "charge_timestamp": 1786206641,
  "firmware_version": "v0.3.1",
  "profile_id": "565318",
  "events": [{ "msec": 0, "type": 0 }],
  "data": {
    "msec": 79000,
    "bt": 118.09, "et": 117.28, "target": 221.57,
    "fan": 81, "heat": 51.48, "rpm": 60,
    "drum_temp": 98.97, "inlet_temp": 223.30, "air_pressure": -55.92,
    "crack": 0, "pcb_temperature": 34.38, "pcb_humidity": 0,
    "temperature_sensors": [
      { "name": "tc0", "value": 98.97 },
      { "name": "tc1", "value": 223.30 },
      { "name": "rtd0", "value": 108.24 },
      { "name": "rtd2", "value": 118.09 }
    ]
  }
}
```

Notes:
- Top-level `batch_uuid`, `charge_timestamp` (unix seconds), `profile_id`, and `firmware_version` identify the roast; `events` mirrors the log event list (`type` 0 = charge, etc.).
- `data` matches the REST datapoint fields, with raw sensors under `temperature_sensors` (`tc*`/`rtd*`) instead of flat `tc0..`/`rtd0..` keys.
- **No `ror`/`ror_float` in the live payload** — Rate of Rise is computed client-side from `bt`/`msec` deltas (unlike the REST datapoint, which includes it).
- `msec` increments in 1000ms steps, matching the ~1/sec cadence.

See `live_roast.py` in the project root for a working monitor (seeds via REST, tails MQTT).
