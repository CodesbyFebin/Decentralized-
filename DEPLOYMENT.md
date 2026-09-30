# Deployment Guide

Complete guide for deploying the Qualification System to production.

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 20.04+), macOS 12+, Windows (WSL2)
- **CPU**: 2+ cores recommended
- **Memory**: 2GB minimum, 4GB+ recommended
- **Disk**: 10GB for base system, 50GB+ for archives
- **Network**: Stable internet connection, port 8080/8081/9090 accessible

### Software Requirements
- Go 1.21+
- PostgreSQL 12+ (or MySQL 8.0+, SQLite for dev)
- Kafka 2.8+ (for event streaming)
- Docker 20.10+ (for containerized deployment)
- Kubernetes 1.24+ (for K8s deployment)

### Required Services
- **Database**: PostgreSQL or MySQL
- **Message Queue**: Kafka or compatible
- **Object Storage**: AWS S3, Google Cloud Storage, or Azure Blob
- **Monitoring**: Prometheus and Grafana (optional)

## Building from Source

### Clone Repository
```bash
git clone https://github.com/CodesbyFebin/Decentralized-.git
cd Decentralized-
```

### Build Binary
```bash
# Build all components
make build

# Build only API service
make build-api

# Build only metrics service
make build-metrics

# Build CLI
make build-cli
```

### Verify Build
```bash
./bin/qualification-system --version
```

## Docker Deployment

### Build Docker Image
```dockerfile
FROM golang:1.21-alpine as builder
WORKDIR /app
COPY . .
RUN apk add --no-cache make
RUN make build

FROM alpine:latest
RUN apk add --no-cache ca-certificates postgresql-client
COPY --from=builder /app/bin/qualification-system /
EXPOSE 8080 9090 8081
CMD ["/qualification-system", "serve"]
```

### Build and Run
```bash
# Build image
docker build -t qualification-system:latest .

# Run container
docker run -d \
  --name qualification \
  -p 8080:8080 \
  -p 9090:9090 \
  -p 8081:8081 \
  -e DB_URL=postgresql://user:pass@db:5432/campaigns \
  -e KAFKA_BROKERS=kafka:9092 \
  qualification-system:latest
```

### Docker Compose
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: campaigns
      POSTGRES_USER: qualif
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  kafka:
    image: confluentinc/cp-kafka:7.0.1
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:7.0.1
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

  api:
    image: qualification-system:latest
    environment:
      DB_URL: postgresql://qualif:secure_password@postgres:5432/campaigns
      KAFKA_BROKERS: kafka:9092
      LISTEN_PORT: 8080
      METRICS_PORT: 9090
      GRAPHQL_PORT: 8081
    ports:
      - "8080:8080"
      - "9090:9090"
      - "8081:8081"
    depends_on:
      - postgres
      - kafka

  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9000:9090"

  grafana:
    image: grafana/grafana
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    ports:
      - "3000:3000"
    depends_on:
      - prometheus

volumes:
  postgres_data:
  prometheus_data:
```

## Kubernetes Deployment

### ConfigMap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: qualification-config
  namespace: default
data:
  db_url: "postgresql://qualif:password@postgres:5432/campaigns"
  kafka_brokers: "kafka:9092"
  s3_bucket: "qualification-backups"
  s3_region: "us-west-2"
```

### Secrets
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: qualification-secrets
  namespace: default
type: Opaque
stringData:
  db_password: "secure_password"
  s3_access_key: "AKIA..."
  s3_secret_key: "..."
  api_key: "sk-..."
```

### Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qualification-system
  namespace: default
  labels:
    app: qualification
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: qualification
  template:
    metadata:
      labels:
        app: qualification
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: api
        image: qualification-system:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        - containerPort: 8081
          name: graphql
        env:
        - name: DB_URL
          valueFrom:
            configMapKeyRef:
              name: qualification-config
              key: db_url
        - name: KAFKA_BROKERS
          valueFrom:
            configMapKeyRef:
              name: qualification-config
              key: kafka_brokers
        - name: S3_BUCKET
          valueFrom:
            configMapKeyRef:
              name: qualification-config
              key: s3_bucket
        - name: S3_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: qualification-secrets
              key: s3_access_key
        - name: S3_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: qualification-secrets
              key: s3_secret_key
        - name: LOG_LEVEL
          value: "info"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        securityContext:
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
      volumes:
      - name: tmp
        emptyDir: {}
      securityContext:
        fsGroup: 1000
```

### Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: qualification-api
  namespace: default
  labels:
    app: qualification
spec:
  type: LoadBalancer
  selector:
    app: qualification
  ports:
  - port: 80
    targetPort: 8080
    name: http
  - port: 9090
    targetPort: 9090
    name: metrics
  - port: 8081
    targetPort: 8081
    name: graphql
```

### Horizontal Pod Autoscaler
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: qualification-hpa
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: qualification-system
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Configuration

### Environment Variables
```bash
# Database
DB_URL=postgresql://user:pass@host:5432/db
DB_CONNECTION_POOL_SIZE=20
DB_MAX_CONNECTIONS=100

# Kafka
KAFKA_BROKERS=kafka1:9092,kafka2:9092,kafka3:9092
KAFKA_SECURITY_PROTOCOL=PLAINTEXT
KAFKA_CONSUMER_GROUP_ID=qualification-system

# Cloud Storage
S3_BUCKET=qualification-backups
S3_REGION=us-west-2
S3_ACCESS_KEY=AKIA...
S3_SECRET_KEY=...

# API Configuration
LISTEN_PORT=8080
METRICS_PORT=9090
GRAPHQL_PORT=8081
SHUTDOWN_TIMEOUT=30s
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s

# Security
API_KEY=sk-...
ENABLE_TLS=true
TLS_CERT_PATH=/etc/tls/cert.pem
TLS_KEY_PATH=/etc/tls/key.pem
ENABLE_MTLS=true

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout

# Performance
CACHE_MAX_SIZE_MB=512
CACHE_MAX_ENTRIES=100000
CACHE_DEFAULT_TTL=5m
QUERY_SLOW_THRESHOLD_MS=100
CONNECTION_POOL_TIMEOUT=5s
```

### Configuration File
```yaml
# config.yaml
database:
  url: postgresql://user:pass@host:5432/db
  pool_size: 20
  max_connections: 100
  connection_timeout: 5s

kafka:
  brokers:
    - kafka1:9092
    - kafka2:9092
  consumer_group: qualification-system
  security:
    protocol: PLAINTEXT

storage:
  backend: s3
  s3:
    bucket: qualification-backups
    region: us-west-2
    access_key: ${S3_ACCESS_KEY}
    secret_key: ${S3_SECRET_KEY}

api:
  listen_port: 8080
  shutdown_timeout: 30s
  rate_limiting:
    enabled: true
    default_limit: 100

performance:
  cache:
    max_size_mb: 512
    eviction_policy: lru
    default_ttl: 5m
  query_optimization:
    slow_query_threshold_ms: 100
    track_patterns: true

monitoring:
  metrics_port: 9090
  prometheus_enabled: true
  grafana_url: http://grafana:3000

logging:
  level: info
  format: json
  output: stdout
```

## Database Setup

### PostgreSQL
```bash
# Create database
createdb campaigns

# Create user
createuser qualif -P  # Prompts for password

# Grant privileges
psql -U postgres -d campaigns -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO qualif;"
```

### Initialize Schema
```bash
./qualification-system db init \
  --db-url postgresql://qualif:pass@localhost/campaigns \
  --create-tables
```

### Run Migrations
```bash
./qualification-system db migrate \
  --db-url postgresql://qualif:pass@localhost/campaigns \
  --version latest
```

## Kafka Setup

### Create Topics
```bash
# Campaign events
kafka-topics --create \
  --topic campaigns.created \
  --partitions 3 \
  --replication-factor 3 \
  --bootstrap-server kafka:9092

# Gate events
kafka-topics --create \
  --topic gates.executed \
  --partitions 3 \
  --replication-factor 3 \
  --bootstrap-server kafka:9092

# Create all topics
./qualification-system kafka init --bootstrap-servers kafka:9092
```

## Monitoring Setup

### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'qualification-system'
    static_configs:
      - targets: ['localhost:9090']
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance
```

### Grafana Dashboards
```bash
# Import dashboard
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @dashboards/qualification-system.json \
  --header "Authorization: Bearer $GRAFANA_TOKEN"
```

## Backup & Recovery

### Enable Automated Backups
```bash
./qualification-system backup init \
  --backend s3 \
  --s3-bucket qualification-backups \
  --schedule "0 2 * * *"  # Daily at 2 AM
```

### Manual Backup
```bash
./qualification-system backup create \
  --db-url postgresql://qualif:pass@localhost/campaigns \
  --output backup-2026-09-30.sql
```

### Restore from Backup
```bash
./qualification-system db restore \
  --db-url postgresql://qualif:pass@localhost/campaigns \
  --backup-file backup-2026-09-30.sql
```

## Health Checks

### Startup Probe
```bash
curl -f http://localhost:8080/health || exit 1
```

### Liveness Probe
```bash
curl -f http://localhost:8080/health || exit 1
```

### Readiness Probe
```bash
curl -f http://localhost:8080/health/ready || exit 1
```

## Troubleshooting

### Service Not Starting
```bash
# Check logs
docker logs qualification-system

# Verify configuration
./qualification-system config check

# Test database connection
./qualification-system db test-connection --db-url postgresql://user:pass@host/db
```

### High Memory Usage
```bash
# Check cache metrics
curl http://localhost:9090/metrics | grep cache

# Reduce cache size
export CACHE_MAX_SIZE_MB=256
```

### Database Connection Issues
```bash
# Verify connection
psql postgresql://qualif:pass@localhost/campaigns -c "SELECT 1;"

# Check connection pool
curl http://localhost:9090/metrics | grep connection_pool
```

### Event Streaming Lag
```bash
# Check Kafka lag
kafka-consumer-groups --bootstrap-server kafka:9092 \
  --group qualification-system \
  --describe

# Increase consumer threads
export KAFKA_CONSUMER_THREADS=4
```

## Production Checklist

- [ ] Database backed up and accessible
- [ ] Kafka cluster configured and reachable
- [ ] S3/cloud storage credentials configured
- [ ] TLS certificates installed and valid
- [ ] Prometheus metrics collection enabled
- [ ] Grafana dashboards configured
- [ ] Logging aggregation enabled (ELK/Splunk)
- [ ] Alerting rules configured
- [ ] Backup/restore tested
- [ ] Disaster recovery plan documented
- [ ] Security scans passed
- [ ] Load testing completed
- [ ] Capacity planning review done
- [ ] Runbooks created for common issues
- [ ] On-call rotation established

## Scaling

### Horizontal Scaling
```bash
# Add more replicas
kubectl scale deployment qualification-system --replicas=5

# Auto-scaling with HPA
kubectl apply -f hpa.yaml
```

### Vertical Scaling
```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2000m"
```

### Database Scaling
- Read replicas for PostgreSQL
- Sharding by tenant_id
- Connection pooling with pgBouncer

## Security Hardening

### Enable TLS
```bash
export ENABLE_TLS=true
export TLS_CERT_PATH=/etc/tls/cert.pem
export TLS_KEY_PATH=/etc/tls/key.pem
```

### Enable mTLS
```bash
export ENABLE_MTLS=true
export CA_CERT_PATH=/etc/tls/ca.pem
```

### API Authentication
```bash
export REQUIRE_API_KEY=true
export API_KEY=sk-...
```

### Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: qualification-network-policy
spec:
  podSelector:
    matchLabels:
      app: qualification
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: default
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 9092  # Kafka
```

## Support

For issues during deployment:
1. Check logs: `docker logs qualification-system`
2. Verify configuration: `./qualification-system config check`
3. Run diagnostics: `./qualification-system doctor`
4. Open GitHub issue: https://github.com/CodesbyFebin/Decentralized-/issues

---

**Last Updated**: 2026-09-30
