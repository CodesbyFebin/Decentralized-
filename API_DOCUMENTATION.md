# API Documentation

Complete reference for REST, GraphQL, and Event Streaming APIs.

## REST API

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication
```bash
# API Key
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/campaigns

# OAuth2 Bearer
curl -H "Authorization: Bearer your-token" http://localhost:8080/api/v1/campaigns
```

### Campaign Endpoints

#### List Campaigns
```http
GET /campaigns
```

**Query Parameters**:
- `status` (optional): Filter by status (pending|running|completed|failed)
- `tenant_id` (optional): Filter by tenant
- `limit` (optional): Number of results (default: 100)
- `offset` (optional): Pagination offset (default: 0)

**Response**:
```json
{
  "data": [
    {
      "id": "camp-1",
      "name": "Q1 2026 Campaign",
      "status": "completed",
      "tenant_id": "tenant-1",
      "created_at": "2026-01-15T10:30:00Z",
      "updated_at": "2026-01-20T15:45:00Z",
      "metrics": {
        "total_gates": 32,
        "passed_gates": 30,
        "failed_gates": 2,
        "success_rate": 93.75
      }
    }
  ],
  "total": 42,
  "limit": 100,
  "offset": 0
}
```

**Rate Limit**: 100 requests/minute  
**Status Codes**: 200 (OK), 400 (Bad Request), 401 (Unauthorized)

#### Create Campaign
```http
POST /campaigns
Content-Type: application/json
```

**Request Body**:
```json
{
  "name": "Q2 2026 Campaign",
  "tenant_id": "tenant-1",
  "metadata": {
    "region": "us-west",
    "environment": "production"
  }
}
```

**Response**:
```json
{
  "id": "camp-2",
  "name": "Q2 2026 Campaign",
  "status": "pending",
  "created_at": "2026-02-01T08:00:00Z"
}
```

**Rate Limit**: 50 requests/minute  
**Status Codes**: 201 (Created), 400 (Bad Request), 401 (Unauthorized), 429 (Too Many Requests)

#### Get Campaign
```http
GET /campaigns/{campaign_id}
```

**Response**:
```json
{
  "id": "camp-1",
  "name": "Q1 2026 Campaign",
  "status": "completed",
  "created_at": "2026-01-15T10:30:00Z",
  "updated_at": "2026-01-20T15:45:00Z",
  "gate_results": [
    {
      "gate_id": "gate-1",
      "passed": true,
      "duration_ms": 145,
      "evidence": {
        "backend": "qemu",
        "node_count": 3
      },
      "executed_at": "2026-01-15T11:00:00Z"
    }
  ],
  "metrics": {
    "total_gates": 32,
    "passed_gates": 30,
    "failed_gates": 2,
    "success_rate": 93.75,
    "average_duration_ms": 156
  }
}
```

**Rate Limit**: 100 requests/minute  
**Status Codes**: 200 (OK), 404 (Not Found)

#### Delete Campaign
```http
DELETE /campaigns/{campaign_id}
```

**Response**: 204 No Content

**Rate Limit**: 20 requests/minute  
**Status Codes**: 204 (No Content), 404 (Not Found), 401 (Unauthorized)

### Metrics Endpoints

#### Get Current Metrics
```http
GET /metrics
```

**Response**:
```json
{
  "timestamp": "2026-09-30T12:00:00Z",
  "campaigns": {
    "total": 42,
    "completed": 38,
    "failed": 2,
    "pending": 2
  },
  "gates": {
    "total_executed": 1344,
    "total_passed": 1250,
    "total_failed": 94,
    "success_rate_percent": 93.01
  },
  "performance": {
    "avg_gate_duration_ms": 156,
    "p95_gate_duration_ms": 235,
    "p99_gate_duration_ms": 450,
    "cache_hit_rate_percent": 78.5
  },
  "system": {
    "memory_alloc_mb": 256,
    "goroutines": 1024,
    "uptime_seconds": 86400
  }
}
```

**Rate Limit**: 200 requests/minute (no auth required)  
**Status Codes**: 200 (OK)

#### Get Metrics History
```http
GET /metrics/history?limit=24&unit=hour
```

**Query Parameters**:
- `limit` (optional): Number of data points (default: 24)
- `unit` (optional): Time unit (hour|day|week, default: hour)

**Response**:
```json
{
  "data_points": [
    {
      "timestamp": "2026-09-29T12:00:00Z",
      "success_rate": 93.5,
      "avg_duration_ms": 154,
      "error_rate": 0.8
    }
  ],
  "period": "24 hours"
}
```

**Rate Limit**: 50 requests/minute  
**Status Codes**: 200 (OK)

### Schema Endpoints

#### Get Current Schema Version
```http
GET /schema/version
```

**Response**:
```json
{
  "version": "1.2.3",
  "applied_at": "2026-09-15T10:00:00Z",
  "hash": "sha256:abcd1234...",
  "migrations_applied": 15,
  "breaking_changes": false
}
```

**Rate Limit**: 200 requests/minute (no auth required)

#### Start Migration
```http
POST /schema/migrate
Content-Type: application/json
```

**Request Body**:
```json
{
  "from_version": "1.2.2",
  "to_version": "1.2.3",
  "strategy": "batch",
  "batch_size": 100,
  "validate": true
}
```

**Response**:
```json
{
  "migration_id": "mig-123",
  "status": "in_progress",
  "progress_percent": 0,
  "started_at": "2026-09-30T12:00:00Z"
}
```

**Rate Limit**: 5 requests/minute

#### Get Migration Status
```http
GET /schema/migrate/{migration_id}
```

**Response**:
```json
{
  "migration_id": "mig-123",
  "status": "completed",
  "progress_percent": 100,
  "started_at": "2026-09-30T12:00:00Z",
  "completed_at": "2026-09-30T12:15:00Z",
  "records_migrated": 1000,
  "validation_result": {
    "passed": true,
    "sample_validated": 10
  }
}
```

**Rate Limit**: 100 requests/minute

### Chaos Testing Endpoints

#### Start Chaos Test
```http
POST /chaos/test
Content-Type: application/json
```

**Request Body**:
```json
{
  "scenario": "database_connection_exhaustion",
  "campaign_id": "camp-1",
  "duration_seconds": 300,
  "severity": "high"
}
```

**Response**:
```json
{
  "test_id": "chaos-123",
  "scenario": "database_connection_exhaustion",
  "status": "running",
  "started_at": "2026-09-30T12:00:00Z"
}
```

**Rate Limit**: 10 requests/minute

#### Get Chaos Test Results
```http
GET /chaos/test/{test_id}
```

**Response**:
```json
{
  "test_id": "chaos-123",
  "scenario": "database_connection_exhaustion",
  "status": "completed",
  "started_at": "2026-09-30T12:00:00Z",
  "completed_at": "2026-09-30T12:05:00Z",
  "duration_seconds": 300,
  "results": {
    "passed": true,
    "invariants_checked": 5,
    "invariants_passed": 5,
    "recovery_time_ms": 1200,
    "max_errors": 15
  }
}
```

**Rate Limit**: 100 requests/minute

### Health Endpoint

#### System Health
```http
GET /health
```

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2026-09-30T12:00:00Z",
  "checks": {
    "database": {
      "status": "healthy",
      "latency_ms": 2
    },
    "kafka": {
      "status": "healthy",
      "brokers": 3
    },
    "cache": {
      "status": "healthy",
      "hit_rate_percent": 78.5
    },
    "memory": {
      "status": "healthy",
      "usage_percent": 45.2
    }
  }
}
```

**Rate Limit**: 1000 requests/minute (no auth required)

## GraphQL API

### Base URL
```
http://localhost:8081/graphql
```

### Schema Introspection
```graphql
query {
  __schema {
    types {
      name
      kind
      fields {
        name
        type {
          name
          kind
        }
      }
    }
  }
}
```

### Queries

#### Get Campaigns
```graphql
query {
  campaigns(limit: 10, status: "completed") {
    id
    name
    status
    createdAt
    metrics {
      totalGates
      passedGates
      failedGates
      successRate
      averageDuration
    }
  }
}
```

#### Get Campaign Details
```graphql
query {
  campaign(id: "camp-1") {
    id
    name
    status
    gateResults {
      gateID
      passed
      duration
      evidence
      executedAt
    }
  }
}
```

#### Get Metrics
```graphql
query {
  metrics {
    timestamp
    campaignsTotal
    campaignsCompleted
    campaignsFailed
    gateSuccessRate
    averageGateDuration
  }
}
```

#### Get Health Status
```graphql
query {
  health {
    status
    timestamp
    checks {
      component
      status
      latency
    }
  }
}
```

### Mutations

#### Create Campaign
```graphql
mutation {
  createCampaign(name: "Q3 2026", tenantID: "tenant-1") {
    id
    name
    status
    createdAt
  }
}
```

#### Start Campaign
```graphql
mutation {
  startCampaign(id: "camp-2") {
    id
    status
    startedAt
  }
}
```

#### Execute Gate
```graphql
mutation {
  executeGate(campaignID: "camp-1", gateID: "gate-1") {
    gateID
    passed
    duration
    evidence
  }
}
```

### Subscriptions

#### Campaign Updates
```graphql
subscription {
  campaignUpdates {
    id
    status
    updatedAt
  }
}
```

#### Gate Results
```graphql
subscription {
  gateResults {
    gateID
    campaignID
    passed
    executedAt
  }
}
```

#### Metrics Updates
```graphql
subscription {
  metricsUpdates {
    timestamp
    successRate
    averageDuration
  }
}
```

## Event Streaming API

### Kafka Integration

**Topic Names**:
- `campaigns.created`
- `campaigns.started`
- `campaigns.completed`
- `campaigns.failed`
- `gates.executed`
- `gates.passed`
- `gates.failed`
- `chaos.started`
- `chaos.ended`
- `migrations.started`
- `migrations.completed`
- `migrations.failed`
- `backups.completed`
- `backups.failed`
- `schema.evolved`
- `metrics.snapshot`

### Message Format
```json
{
  "id": "event-123",
  "type": "campaign.created",
  "severity": "info",
  "source": "campaign",
  "timestamp": "2026-09-30T12:00:00Z",
  "campaign_id": "camp-1",
  "subject": "Campaign Q1 2026 created",
  "description": "New campaign for Q1 2026 qualification",
  "metadata": {
    "tenant_id": "tenant-1",
    "region": "us-west"
  },
  "trace_id": "trace-123"
}
```

### Subscribe Example (Go)
```go
filter := &EventFilter{
  EventTypes: []EventType{
    EventTypeCampaignCreated,
    EventTypeCampaignCompleted,
  },
}

sub, err := eventStream.Subscribe("my-sub", filter, func(event *StreamEvent) error {
  log.Printf("Event: %s", event.Type)
  return nil
})
```

## Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Campaign not found",
    "details": {
      "campaign_id": "camp-999"
    },
    "timestamp": "2026-09-30T12:00:00Z"
  }
}
```

### Error Codes
| Code | Status | Description |
|------|--------|-------------|
| RESOURCE_NOT_FOUND | 404 | Requested resource not found |
| INVALID_REQUEST | 400 | Request validation failed |
| UNAUTHORIZED | 401 | Authentication required |
| FORBIDDEN | 403 | Insufficient permissions |
| TOO_MANY_REQUESTS | 429 | Rate limit exceeded |
| INTERNAL_ERROR | 500 | Server error |
| SERVICE_UNAVAILABLE | 503 | Service temporarily unavailable |

## Rate Limiting

### Headers
- `X-RateLimit-Limit`: Max requests allowed
- `X-RateLimit-Remaining`: Requests remaining
- `X-RateLimit-Reset`: Timestamp when limit resets

### Example
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 42
X-RateLimit-Reset: 1696056060
```

### Handling Rate Limits
```bash
# Retry with exponential backoff
curl -w "Status: %{http_code}" \
  --retry 3 \
  --retry-delay 1 \
  http://localhost:8080/api/v1/campaigns
```

## Pagination

### Query Parameters
- `limit`: Number of results (default: 100, max: 1000)
- `offset`: Pagination offset (default: 0)

### Response Format
```json
{
  "data": [],
  "total": 500,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

---

**API Version**: 1.0.0  
**Last Updated**: 2026-09-30
