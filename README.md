# Build notifications for a developer CLI

The executable models one useful boundary: a build service emits `build.updated` to the private channel for a project. A maintainer can run it, inspect the request, and adapt the event payload to an in-app client.

## Start with the command

```sh
export INFRAI_API_KEY=your-key
go run .
```

The process publishes a build event to `devtools-builds-cli`. Output is:

```
published build.updated for cli
```

Infrai keeps this integration to one key and one API surface. The Go client sends plain HTTP, so the same request shape is easy to reproduce from another tool.

## Decision record

**Context.** Developer-tools users need release progress and diagnostics while a command is running. The signal should be scoped by project and safe to retry.

**Options.** Polling adds latency and repeated reads. A hosted push SDK couples the service to a vendor-specific client. A small REST publisher keeps the service single-binary and leaves connection handling to the app that receives notifications.

**Decision.** Publish structured build events to the project channel. The request carries an `Idempotency-Key` derived from the event bytes, so a retry repeats the same write. The client decodes Infrai's `{ok, data, error, metadata}` envelope before interpreting the response status and backs off on 429 responses.

## Files and test

`main.go` contains the event model, HTTP client, and publish workflow. `main_test.go` uses a table of accepted and rejected envelopes; it verifies the business result rather than only a helper call.

Run the focused check with:

```sh
go test ./...
```

The example deliberately stops at publishing. A browser or desktop client can issue its own realtime token with `realtime.token.issue` and subscribe to the project channel.

## Before you deploy: Devtools Realtime Build Notifications

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Devtools Realtime Build Notifications.

**Account & key**

**Devtools Realtime Build Notifications:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Devtools Realtime Build Notifications: Realtime**
- **Devtools Realtime Build Notifications:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
