#!/bin/bash

BASE_URL="http://localhost:8080"

# 1. Login
LOGIN_RESP=$(curl --noproxy "*" -s -X POST "$BASE_URL/api/v1/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ Login Failed"
    exit 1
fi
echo "✅ Login Success"

# 2. Add Local Host
echo "Adding Local Host..."
ADD_RESP=$(curl --noproxy "*" -s -X POST "$BASE_URL/api/v1/docker/hosts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Local-Test-Shell",
    "host_id": "local"
  }')
echo "Add Response: $ADD_RESP"

HOST_ID=$(echo $ADD_RESP | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Host ID: $HOST_ID"

if [ -z "$HOST_ID" ]; then
    echo "❌ Failed to add host"
    exit 1
fi

# 3. Sync/Get Info
echo "Triggering Sync (GetInfo)..."
INFO_RESP=$(curl --noproxy "*" -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/docker/hosts/$HOST_ID/info")
echo "Info Response: $INFO_RESP"

# 4. List Containers
echo "Listing Containers..."
LIST_RESP=$(curl --noproxy "*" -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/docker/hosts/$HOST_ID/containers")
echo "List Response: $LIST_RESP"

# 5. Diagnosis
echo "Running Diagnosis..."
DIAG_RESP=$(curl --noproxy "*" -s -X POST -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/docker/hosts/$HOST_ID/test")
echo "Diag Response: $DIAG_RESP"
