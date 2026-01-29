#!/bin/bash

# Test script for CMDB Host Creation
BASE_URL="http://localhost:8080"

# 1. Login
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ Login Failed"
    exit 1
fi
echo "✅ Login Success"

# 2. Test CMDB Create
echo "Testing CMDB Create Host..."
CREATE_RESP=$(curl -s -X POST "$BASE_URL/api/v1/cmdb/hosts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{ 
    "name": "test-host-01",
    "ip": "192.168.1.100",
    "port": 22,
    "username": "root",
    "group_name": "test-group"
  }')

echo "Response: $CREATE_RESP"

if echo "$CREATE_RESP" | grep -q '"code":0'; then
    echo "✅ Create Host Success"
else
    echo "❌ Create Host Failed"
fi
