# Database Setup Guide

## Overview

The Decentralized.Host system uses PostgreSQL 16+ for persistent storage. This guide covers:
- Local development setup
- Production deployment
- Schema initialization
- Test data fixtures
- Backup and recovery

## Local Development Setup

### Prerequisites

- PostgreSQL 16 or higher
- `psql` command-line client
- `createdb` utility

### Installation

#### macOS (Homebrew)
```bash
brew install postgresql@16
brew services start postgresql@16
```

#### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql
```

#### Windows
Download and install from https://www.postgresql.org/download/windows/

### Create Development Database

```bash
# Connect to PostgreSQL default database
psql -U postgres

# Create database and user
CREATE DATABASE decentralized_host;
CREATE USER dev_user WITH PASSWORD 'dev-password-change-me';
ALTER ROLE dev_user WITH CREATEDB;
GRANT ALL PRIVILEGES ON DATABASE decentralized_host TO dev_user;
\q
```

### Initialize Schema

```bash
# From repository root
psql -U dev_user -d decentralized_host -f db/schema.sql

# Verify tables were created
psql -U dev_user -d decentralized_host -c "\dt"
```

### Update Environment Variables

Create `.env` file in `api/` directory:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=dev_user
DB_PASSWORD=dev-password-change-me
DB_NAME=decentralized_host

# Server
PORT=3001
NODE_ENV=development

# JWT
JWT_SECRET=dev-secret-key-change-in-production

# CORS
CORS_ORIGIN=http://localhost:3000

# Mail Server
MAIL_SMTP_PORT=25
MAIL_SMTP_TLS_PORT=587
MAIL_IMAP_PORT=993
MAIL_IMAP_PLAIN_PORT=143
MAIL_FROM_DOMAIN=localhost
MAIL_ENCRYPTION_KEY=dev-encryption-key-change-in-production
MAIL_ATTACHMENT_STORAGE=/tmp/mail_attachments
MAX_ATTACHMENT_SIZE_MB=25
REDIS_URL=redis://localhost:6379
```

## Docker Compose Setup

### Start All Services

```bash
docker-compose up -d
```

This will start:
- PostgreSQL (port 5432)
- Redis (port 6379)
- API server (port 3001)
- Web frontend (port 3000)

Schema is automatically initialized via `db/schema.sql` mounted in the postgres service.

### Verify Services

```bash
# Check container status
docker-compose ps

# View logs
docker-compose logs -f api

# Connect to database
psql -U postgres -h localhost -d decentralized_host
```

### Stop Services

```bash
docker-compose down
```

## Schema Overview

### Core Tables

| Table | Purpose |
|-------|---------|
| `users` | User accounts and authentication |
| `teams` | Team/organization management |
| `team_members` | Team membership and roles |
| `api_keys` | API authentication keys |
| `audit_logs` | Activity logging |

### Infrastructure Tables

| Table | Purpose |
|-------|---------|
| `nodes` | Compute nodes in the cluster |
| `deployments` | Application deployments |
| `alerts` | System alerts and monitoring |

### Mail Server Tables

| Table | Purpose |
|-------|---------|
| `mailboxes` | Email accounts |
| `emails` | Email messages |
| `email_folders` | Folder structure (Inbox, Sent, Drafts, etc.) |
| `email_attachments` | Email file attachments |
| `email_queue` | Outbound email delivery queue |
| `mail_settings` | Per-mailbox configuration |

## Test Data Setup

### Create Test User

```bash
psql -U dev_user -d decentralized_host << 'EOF'
INSERT INTO users (email, password_hash, full_name, role)
VALUES (
  'testuser@example.com',
  '$2b$10$YOUR_BCRYPT_HASH_HERE',
  'Test User',
  'user'
);
EOF
```

### Create Test Team

```bash
psql -U dev_user -d decentralized_host << 'EOF'
INSERT INTO teams (name, description, owner_id)
SELECT 'Test Team', 'Test team for development', id
FROM users WHERE email = 'testuser@example.com';
EOF
```

### Create Test Mailbox

```bash
psql -U dev_user -d decentralized_host << 'EOF'
INSERT INTO mailboxes (user_id, email_address, display_name, storage_quota_mb)
SELECT id, 'test@example.com', 'Test Mailbox', 5120
FROM users WHERE email = 'testuser@example.com';
EOF
```

## Integration Testing

### Prerequisites

- API server running (`npm run dev` in `api/` directory)
- Database initialized
- Test user created

### Run Mail API Tests

```bash
# Test mailbox creation
curl -X POST http://localhost:3001/api/v1/mail/mailbox \
  -H 'Authorization: Bearer YOUR_JWT_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "email_address": "newmail@example.com",
    "display_name": "New Mailbox",
    "password": "secure-password-123"
  }'

# Test getting folders
curl -X GET http://localhost:3001/api/v1/mail/mailbox/MAILBOX_ID/folders \
  -H 'Authorization: Bearer YOUR_JWT_TOKEN'

# Test sending email
curl -X POST http://localhost:3001/api/v1/mail/send \
  -H 'Authorization: Bearer YOUR_JWT_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "to_addresses": ["recipient@example.com"],
    "subject": "Test Email",
    "body_text": "This is a test email"
  }'
```

## Production Deployment

### Prerequisites

- PostgreSQL 16+ with SSL/TLS
- Automated backups configured
- Connection pooling (PgBouncer recommended)
- Monitoring and alerting

### Connection String

```
postgresql://username:password@host:5432/decentralized_host?sslmode=require
```

### Security Best Practices

1. **Use Strong Passwords**
   ```bash
   # Generate strong password
   openssl rand -base64 32
   ```

2. **Enable SSL/TLS**
   ```bash
   # In postgresql.conf
   ssl = on
   ssl_cert_file = '/path/to/cert.pem'
   ssl_key_file = '/path/to/key.pem'
   ```

3. **Restrict Connections**
   ```bash
   # In pg_hba.conf
   host  decentralized_host  app_user  10.0.0.0/8  md5
   ```

4. **Regular Backups**
   ```bash
   # Daily backup script
   pg_dump -U postgres decentralized_host | gzip > backup_$(date +%Y%m%d).sql.gz
   ```

### Connection Pooling with PgBouncer

```ini
# /etc/pgbouncer/pgbouncer.ini
[databases]
decentralized_host = host=localhost port=5432 dbname=decentralized_host

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
min_pool_size = 10
```

## Maintenance

### Regular Tasks

- **Daily**: Monitor database size and performance
- **Weekly**: Verify backup integrity
- **Monthly**: Analyze query performance and index usage
- **Quarterly**: Review and optimize slow queries

### Useful Commands

```bash
# Database size
psql -U postgres -c "SELECT pg_size_pretty(pg_database_size('decentralized_host'));"

# Table sizes
psql -U postgres -d decentralized_host -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) FROM pg_tables WHERE schemaname != 'pg_catalog' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

# Active connections
psql -U postgres -c "SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;"

# Vacuum and analyze
psql -U postgres -d decentralized_host -c "VACUUM ANALYZE;"
```

## Troubleshooting

### Connection Refused

```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# View PostgreSQL logs
tail -f /var/log/postgresql/postgresql.log
```

### Permission Denied

```bash
# Check user permissions
psql -U postgres -d decentralized_host -c "\du"

# Grant privileges
psql -U postgres -d decentralized_host -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO dev_user;"
```

### Slow Queries

```bash
# Enable query logging
psql -U postgres -c "ALTER SYSTEM SET log_min_duration_statement = 1000;"
psql -U postgres -c "SELECT pg_reload_conf();"

# View slow queries
SELECT query, calls, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;
```

## Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Docker Compose Docs](https://docs.docker.com/compose/)
- [Schema Migration Guide](./MIGRATIONS.md)
