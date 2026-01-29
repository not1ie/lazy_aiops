#!/bin/bash

# Test script to verify core functionality inside the container environment
BASE_URL="http://localhost:8080"

echo "1. Testing Health..."
curl -s "$BASE_URL/health"
echo -e "\n"

echo "2. Testing Login..."
# 尝试登录获取 Token
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}')
echo "Response: $LOGIN_RESP"
TOKEN=$(echo $LOGIN_RESP | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ Login Failed"
    exit 1
fi
echo "✅ Login Success, Token: ${TOKEN:0:10}..."

echo "3. Testing System Info..."
curl -s "$BASE_URL/api/v1/system/info"
echo -e "\n"

echo "4. Testing AI Chat (Mock)..."
curl -s -X POST "$BASE_URL/api/v1/ai/chat" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello"}'
echo -e "\n"

echo "5. Testing Knowledge Base List..."
curl -s "$BASE_URL/api/v1/knowledge/docs" \
  -H "Authorization: Bearer $TOKEN"
echo -e "\n"
