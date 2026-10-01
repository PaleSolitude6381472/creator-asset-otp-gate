#!/bin/sh
set -eu

: "${CREATOR_PHONE:?set CREATOR_PHONE to an E.164 phone number}"
: "${OTP_CODE:?set OTP_CODE to the received code}"

payload=$(printf '{"creator_id":"creator-7","phone":"%s","asset_id":"preset-pack","subscriber_version":4,"current_version":4,"processing_state":"ready","attempt_id":"demo-42"}' "$CREATOR_PHONE")

curl --fail-with-body --silent --show-error -X POST http://127.0.0.1:8080/access/start \
  -H 'Content-Type: application/json' -d "$payload"
printf '\n'
curl --fail-with-body --silent --show-error -X POST http://127.0.0.1:8080/access/verify \
  -H 'Content-Type: application/json' \
  -d "${payload%?},\"code\":\"$OTP_CODE\"}"
printf '\n'
