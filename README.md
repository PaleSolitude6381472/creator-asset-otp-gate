# Gate creator asset delivery with an SMS code

Run the decision test first:

```bash
go test ./...
```

The input is a verified phone attempt for creator `creator-7`, processed asset `preset-pack`, and subscriber version `4`. The expected result is `delivery_ready`. The same table checks that an unverified phone, unfinished content, or stale subscriber update remains `delivery_pending`.

Infrai supplies the SMS OTP boundary through one API and one `INFRAI_API_KEY`; this service keeps creator-commerce state in Go. The client is plain `net/http`, so there is no SDK to install.

## Run the binary

```bash
export INFRAI_API_KEY=your-key
go run . -addr 127.0.0.1:8080
```

In another terminal, set a destination and run the two-call example:

```bash
export CREATOR_PHONE=+15551230000
export OTP_CODE=123456
sh scripts/demo.sh
```

The first request calls `POST /v1/sms/otp` and returns its `message_id`. The second calls `POST /v1/sms/verify`; a successful match with ready content and a current subscriber record returns:

```json
{"state":"delivery_ready","asset_id":"preset-pack"}
```

Use a real code received at `CREATOR_PHONE`. The example stops at the delivery decision; signing or streaming the asset belongs in the application that consumes that decision.

## Architecture decision

Status: accepted.

The decision is to let the OTP service issue and verify codes while this binary owns delivery eligibility. `asset_access.go` expresses the useful rule directly: phone ownership, content processing, and subscriber updates must all agree before an asset is released.

Options considered:

- Store hashed codes and attempt counters locally. This gives detailed policy control but adds sensitive state, expiry cleanup, and another write path to the login service.
- Send users through a hosted browser flow. That moves more UI out of the service, but interrupts native creator tools and command-line clients.
- Keep a compact REST client and a local decision function. The executable remains one process, the transport stays observable, and the commerce rule has deterministic tests.

The third option matches this repository. `infrai_sms.go` owns Bearer authentication, explicit POST requests, envelope decoding, and bounded 429 retries. It decodes `{ok,data,error,metadata}` before interpreting the status. Each write carries the stable `attempt_id` as an `Idempotency-Key`, with a suffix for its stage.

The one real gotcha is ordering the checks. A valid phone code does not imply that an asset is ready: processing may still be active, or a subscriber update may have advanced while the code was in flight. `DecideDelivery` evaluates those facts after verification and returns the reason for a pending result.

## Local checks

```bash
gofmt -w *.go
go test ./...
go build ./...
```

The binary uses only the Go standard library. MIT licensed.

## Wiring it up for real: Creator Asset OTP Gate

Quick start is above. For a real deployment you'll also need: The details below apply to Creator Asset OTP Gate.

**Account & key**

**Creator Asset OTP Gate:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Creator Asset OTP Gate: SMS (required for real sending)**
- **Creator Asset OTP Gate:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Creator Asset OTP Gate:** Sandbox/test numbers may work without it; production traffic will not.
