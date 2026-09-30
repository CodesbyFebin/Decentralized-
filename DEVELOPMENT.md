# Decentralized.Host Development Guide

## Project Overview

This is a full-stack command centre for Decentralized.Host, a sovereign infrastructure platform with signed intent, local policy enforcement, and explicit state management.

### Architecture

**Frontend:** Next.js 14 (React 18, TypeScript, Tailwind CSS)
- 30 pages implementing the complete Command Centre UI
- Dark theme with consistent component library
- Recharts for data visualization
- Next.js 14 static export for deployment

**Backend:** Express.js with PostgreSQL
- RESTful API with JWT authentication
- User and team management
- Resources: teams, nodes, deployments, alerts, audit logs
- PostgreSQL database with schema migrations

**Infrastructure:** Docker Compose
- PostgreSQL 16 Alpine container
- Express.js API service
- Next.js frontend service
- Development and production configurations

## Installation & Setup

### Prerequisites
- Node.js v18+ (v22.22.2 tested)
- npm v10+
- Docker and Docker Compose (for full stack)
- PostgreSQL 16+ (for database)

### Frontend Setup

```bash
cd web
npm install
npm run dev       # Start dev server on http://localhost:3000
npm run build     # Build for production
```

### Backend Setup

```bash
cd api
npm install --legacy-peer-deps
cp .env.example .env  # Configure environment variables
npm run dev       # Start dev server on http://localhost:3001
npm run build     # Build TypeScript
npm run test      # Run tests
```

### Full Stack with Docker

```bash
docker compose up -d
# Access frontend: http://localhost:3000
# Access API: http://localhost:3001
# PostgreSQL: localhost:5432
```

## Project Structure

### Frontend (`/web`)
```
web/
├── src/
│   ├── pages/
│   │   ├── login/          # Authentication
│   │   ├── register/
│   │   ├── dashboard/      # Main dashboard
│   │   ├── nodes/          # Infrastructure management
│   │   ├── deployments/
│   │   ├── storage/
│   │   ├── analytics/      # Metrics and analytics
│   │   ├── alerts/         # Alert management
│   │   ├── incidents/      # Incident tracking
│   │   ├── security/       # Security policies
│   │   ├── compliance/     # Compliance tracking
│   │   ├── audit/          # Audit logs
│   │   ├── integrations/   # Third-party integrations
│   │   └── ...
│   ├── components/         # Reusable React components
│   ├── layouts/           # Page layouts
│   ├── lib/
│   │   ├── api.ts         # API client
│   │   └── withAuth.tsx   # Protected route HOC
│   └── types/             # TypeScript type definitions
├── tailwind.config.ts     # Tailwind CSS configuration
└── tsconfig.json
```

### Backend (`/api`)
```
api/
├── src/
│   ├── index.ts           # Express app initialization
│   ├── middleware/
│   │   └── auth.ts        # JWT authentication
│   └── routes/
│       ├── auth.ts        # Login/register
│       ├── users.ts       # User management
│       ├── teams.ts       # Team CRUD operations
│       ├── nodes.ts       # Node management
│       ├── alerts.ts      # Alert management
│       └── audit.ts       # Audit logs
├── package.json
├── tsconfig.json
└── .env                   # Environment configuration
```

### Database (`/db`)
```
db/
└── schema.sql             # PostgreSQL schema definition
```

## API Endpoints

### Authentication
- `POST /auth/register` - Create new account
- `POST /auth/login` - Sign in with email/password

### Users
- `GET /users/me` - Get current user
- `GET /users` - List all users

### Teams
- `GET /teams` - List user's teams
- `GET /teams/:id` - Get team details
- `POST /teams` - Create team
- `PATCH /teams/:id` - Update team
- `DELETE /teams/:id` - Delete team

### Nodes
- `GET /nodes` - List nodes
- `GET /nodes/:id` - Get node details
- `POST /nodes` - Create node
- `PATCH /nodes/:id` - Update node

### Alerts
- `GET /alerts` - List alerts (supports filtering by severity/status)
- `GET /alerts/:id` - Get alert details
- `POST /alerts` - Create alert
- `PATCH /alerts/:id` - Update alert

### Audit
- `GET /audit` - List audit logs (with pagination)
- `GET /audit/:id` - Get audit log details
- `POST /audit` - Create audit log

## Authentication

The system uses JWT tokens for authentication:

1. **Register:** User creates account with email/password
2. **Login:** User receives JWT token on successful login
3. **Storage:** Token stored in localStorage
4. **Protected Routes:** Dashboard and resource pages require valid token
5. **API Requests:** Token automatically included in `Authorization: Bearer <token>` header

## Frontend Pages (30 Total)

### Core Infrastructure
- Dashboard - Main overview with metrics and recent activity
- Nodes - Node management and monitoring
- Compute - Compute resources allocation
- Storage - Storage bucket management
- Deployments - Workload deployment tracking

### Management & Security
- Domains - Domain and DNS management
- Security - Security policies and certificates
- Capacity - Resource capacity planning

### Operations & Monitoring
- Status - System health and status
- Analytics - Detailed metrics and trends
- Alerts - Alert creation and management
- Logs - Application and system logs
- Incidents - Incident tracking and MTTR

### Administration
- Billing - Billing and usage tracking
- Network - Network configuration
- Backups - Backup management and recovery
- Database - Database administration
- API Docs - API documentation

### Integrations & Settings
- Integrations - Third-party service integrations
- Webhooks - Webhook management
- Team - Team member management
- Settings - System settings
- Compliance - Compliance framework tracking
- Postmortems - Post-incident analysis
- Audit - Audit log review
- Copilot - AI assistant
- Evidence - Evidence collection for qualification

## CI/CD Pipeline

GitHub Actions workflow (`.github/workflows/build-and-test.yml`):
- Runs on Node 18.x and 20.x
- Linting and type-checking
- Build verification
- Artifact upload

## Next Steps

### Immediate (P0)
1. Set up actual PostgreSQL database for development
2. Implement database migrations and seeding
3. Connect frontend API calls to real backend endpoints
4. Test full authentication flow end-to-end
5. Deploy to production environment

### Short Term (P1)
1. Add email notifications for alerts and incidents
2. Implement WebSocket support for real-time updates
3. Add multi-factor authentication (MFA)
4. Implement role-based access control (RBAC) on frontend
5. Add comprehensive API error handling

### Medium Term (P2)
1. Implement backend task queue for async operations
2. Add search and filtering to all list pages
3. Implement analytics data generation and collection
4. Add webhook event delivery and retry logic
5. Implement backup and disaster recovery features

### Long Term (P3)
1. Kubernetes integration for orchestration
2. Advanced compliance reporting
3. Machine learning for anomaly detection
4. Advanced audit trail analysis
5. Disaster recovery and business continuity features

## Troubleshooting

### Build Errors
- Clear node_modules: `rm -rf node_modules && npm install`
- Clear Next.js cache: `rm -rf .next`
- Check Node version: `node --version`

### Database Connection Errors
- Verify PostgreSQL is running
- Check DB credentials in `.env`
- Verify database exists: `psql -h localhost -U postgres`

### Port Already in Use
- Find process: `lsof -i :3000` or `lsof -i :3001`
- Kill process: `kill -9 <PID>`
- Or change port in `.env`

### API CORS Errors
- Verify `CORS_ORIGIN` in backend `.env`
- Check request headers include origin
- Clear browser cache

## Development Tips

### Adding New Pages
1. Create page file in `/web/src/pages/`
2. Use AppLayout wrapper for consistent UI
3. Import withAuth HOC for protected pages
4. Add to Sidebar navigation if needed

### Adding API Endpoints
1. Create route file in `/api/src/routes/`
2. Register route in `/api/src/index.ts`
3. Add authentication middleware if needed
4. Update API client in frontend if needed

### Testing
- Backend: `npm run test` (Jest configured)
- Frontend: No test setup yet - manual testing recommended
- End-to-end: Use actual browser to test flows

## Environment Variables

### Frontend (`/web/.env.local`)
- `NEXT_PUBLIC_API_URL` - Backend API URL (default: `/api/v1`)

### Backend (`/api/.env`)
- `PORT` - API server port (default: 3001)
- `NODE_ENV` - Environment (development/production)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `JWT_SECRET` - Secret key for JWT tokens
- `CORS_ORIGIN` - Allowed CORS origin (default: http://localhost:3000)

## Contact & Support

For issues or questions, refer to:
- GitHub Issues: https://github.com/CodesbyFebin/Decentralized-
- AGENTS.md: Project claims and verification methods
- CLAUDE.md: Claude development guidelines (if present)

## License

See LICENSE file for details.
