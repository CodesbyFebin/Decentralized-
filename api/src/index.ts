import express, { Request, Response, NextFunction } from 'express'
import cors from 'cors'
import morgan from 'morgan'
import dotenv from 'dotenv'
import { Pool } from 'pg'
import authRoutes from './routes/auth'
import usersRoutes from './routes/users'
import teamsRoutes from './routes/teams'
import nodesRoutes from './routes/nodes'
import alertsRoutes from './routes/alerts'
import auditRoutes from './routes/audit'
import { createMailRoutes } from './routes/mail'
import { authenticateToken } from './middleware/auth'

dotenv.config()

const app = express()
const PORT = process.env.PORT || 3001

// Database connection pool
export const pool = new Pool({
  user: process.env.DB_USER || 'postgres',
  password: process.env.DB_PASSWORD || 'postgres',
  host: process.env.DB_HOST || 'localhost',
  port: parseInt(process.env.DB_PORT || '5432'),
  database: process.env.DB_NAME || 'decentralized_host',
})

// Middleware
app.use(cors())
app.use(morgan('combined'))
app.use(express.json())

// Health check
app.get('/health', (req: Request, res: Response) => {
  res.json({ status: 'healthy', timestamp: new Date().toISOString() })
})

// Public routes (no authentication required)
app.use('/api/v1/auth', authRoutes)

// Protected routes (authentication required)
app.use('/api/v1/users', authenticateToken, usersRoutes)
app.use('/api/v1/teams', authenticateToken, teamsRoutes)
app.use('/api/v1/nodes', authenticateToken, nodesRoutes)
app.use('/api/v1/alerts', authenticateToken, alertsRoutes)
app.use('/api/v1/audit', authenticateToken, auditRoutes)
app.use('/api/v1/mail', createMailRoutes(pool))

// Error handling middleware
app.use((err: any, req: Request, res: Response, next: NextFunction) => {
  console.error(err)
  res.status(err.status || 500).json({
    success: false,
    error: err.message || 'Internal server error'
  })
})

// 404 handler
app.use((req: Request, res: Response) => {
  res.status(404).json({
    success: false,
    error: 'Route not found'
  })
})

// Start server
app.listen(PORT, () => {
  console.log(`🚀 API Server running on port ${PORT}`)
})
