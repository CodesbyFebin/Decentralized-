#!/bin/bash
set -e
BASE_URL="http://localhost:3001/api/v1"
PASSED=0
FAILED=0

echo "🧪 Testing Mail Server"
echo "======================================"

# Test health
HEALTH=$(curl -s "$BASE_URL/health")
if echo "$HEALTH" | grep -q "ok"; then
  echo "✅ Health check passed"
  ((PASSED++))
else
  echo "❌ Health check failed"
  ((FAILED++))
fi

# Test create mailbox
MAILBOX=$(curl -s -X POST "$BASE_URL/mailboxes" -H "Content-Type: application/json" -d '{"username":"testuser2","password":"TestPass123","name":"Test"}')
TOKEN=$(echo "$MAILBOX" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
MAILBOX_ID=$(echo "$MAILBOX" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
  echo "✅ Mailbox created"
  ((PASSED++))
else
  echo "❌ Mailbox creation failed"
  ((FAILED++))
fi

# Test get mailbox
GET_MB=$(curl -s "$BASE_URL/mailboxes/$MAILBOX_ID" -H "Authorization: Bearer $TOKEN")
if echo "$GET_MB" | grep -q "Test"; then
  echo "✅ Get mailbox passed"
  ((PASSED++))
else
  echo "❌ Get mailbox failed"
  ((FAILED++))
fi

# Test create folder
FOLDER=$(curl -s -X POST "$BASE_URL/mailboxes/$MAILBOX_ID/folders" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"Test Folder"}')
if echo "$FOLDER" | grep -q "Test Folder"; then
  echo "✅ Create folder passed"
  ((PASSED++))
else
  echo "❌ Create folder failed"
  ((FAILED++))
fi

# Test list emails
EMAILS=$(curl -s "$BASE_URL/mailboxes/$MAILBOX_ID/emails" -H "Authorization: Bearer $TOKEN")
if echo "$EMAILS" | grep -q '"data"'; then
  echo "✅ List emails passed"
  ((PASSED++))
else
  echo "❌ List emails failed"
  ((FAILED++))
fi

# Test queue status
QUEUE=$(curl -s "$BASE_URL/queue/status" -H "Authorization: Bearer $TOKEN")
if echo "$QUEUE" | grep -q "pending"; then
  echo "✅ Queue status passed"
  ((PASSED++))
else
  echo "❌ Queue status failed"
  ((FAILED++))
fi

# Test delete mailbox
DELETE=$(curl -s -X DELETE "$BASE_URL/mailboxes/$MAILBOX_ID" -H "Authorization: Bearer $TOKEN")
if echo "$DELETE" | grep -q "success"; then
  echo "✅ Delete mailbox passed"
  ((PASSED++))
else
  echo "❌ Delete mailbox failed"
  ((FAILED++))
fi

echo ""
echo "======================================"
echo "✅ Passed: $PASSED | ❌ Failed: $FAILED"
