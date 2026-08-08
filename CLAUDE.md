# CLAUDE.md

Guidance for Claude Code (and other AI agents) working in this repository.

## What this is

`roestat` is a terminal UI (Bubble Tea) for [ROEST](https://roestcoffee.com)
sample coffee roasters. It has two views: a **live roast dashboard** fed by an
MQTT-over-WebSocket stream, and a **roast history browser** (list → detail) fed
by the ROEST REST API. It is an unofficial client of ROEST's undocumented cloud
API.

## Commands

```sh
go build -o roestat .   # build the binary
go test ./...           # tests (incl. a headless render test of every view)
go vet ./...
gofmt -w .              # format before committing
./roestat               # run (needs credentials — see below)
```

Requires Go 1.24+. Keep the code `gofmt`-clean and `go vet`-clean.

## Configuration / running

Credentials come from the environment or a git-ignored `.env` (see
`.env.example`): `ROEST_CLIENT_ID`, `ROEST_CLIENT_SECRET`, and optional
`ROEST_CUSTOMER_ID` (auto-detected from the account if unset). **Never commit
`.env` or any real credentials, customer IDs, or roast data.**

## Architecture

```
main.go                  # entrypoint: load .env, build client, start Bubble Tea
internal/roest/          # ROEST API client (no UI deps)
  client.go              #   OAuth token, HTTP helper, sanitizeJSON, customer-ID resolution
  machines.go logs.go datapoints.go   # REST endpoints
  live.go                #   MQTT-over-WSS subscription -> LivePayload channel
  ror.go                 #   Rate-of-Rise (°C/min) from bean-temp deltas
internal/ui/             # Bubble Tea models (import roest, never the reverse)
  app.go                 #   root model: view enum, key routing, navigation
  picker.go history.go detail.go live.go   # the views
  chart.go               #   shared ntcharts time-series line chart (braille)
  messages.go            #   tea.Msg types + async command constructors
  styles.go widgets.go   #   lipgloss styling + stat/legend helpers
```

Data flow: `internal/roest` is a plain API client that knows nothing about the
UI. `internal/ui` calls it via `tea.Cmd`s in `messages.go` (async, off the render
loop). The live feed pushes `LivePayload`s onto a channel; `waitForLive` turns
each into a `liveDataMsg` and is re-issued after every message to keep the stream
flowing.

## Domain gotchas (important, non-obvious)

- **RoR is computed client-side.** The live MQTT feed never includes Rate of
  Rise, and REST datapoints frequently return it as null. Both the live and
  detail views derive it from bean-temp deltas over a 30s window (`roest.RoR`).
- **`/datapoints/?page_size=all` returns a bare JSON array**, not the usual
  paginated envelope (empty = `[]`). The client parses both shapes.
- **JSON control chars.** The API emits raw control characters inside string
  values. All response bodies (and MQTT payloads) go through `sanitizeJSON`
  before `json.Unmarshal`; keep using it for new endpoints.
- **Live payload shape differs from REST.** MQTT messages nest sensor data under
  `data`, with raw sensors in a `temperature_sensors[]` array of `{name,value}`
  (not flat `tc0`/`rtd0` keys).
- **Nullable fields are pointers.** Many log/datapoint fields (bean name,
  weights, temps, RoR) can be null; model them as pointers and guard before use.

## Charts

Use `internal/ui/chart.go`'s `renderChart` for any curve (it draws braille
time-series lines via ntcharts, with an elapsed `m:ss` X axis). X is elapsed
seconds, Y is the value. Do not reintroduce `wavelinechart` — it rendered roast
curves as vertical bars, which is why the chart uses `timeserieslinechart`.

## Conventions

- Match the surrounding style; keep the two packages' dependency direction
  one-way (`ui` → `roest`).
- When adding a view, add its state to the `viewState` enum and route keys in
  `app.go`; add async work as a `tea.Cmd` in `messages.go`.
- Update `README.md` when you change flags, env vars, or keybindings.
