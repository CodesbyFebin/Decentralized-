import { Router, Request, Response } from 'express'
import { AuthRequest } from '../middleware/auth'
import { pool } from '../index'

const router = Router()

// Get all audit logs for user's teams
router.get('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const { action, limit = 100, offset = 0 } = req.query

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    let query = `
      SELECT al.id, al.team_id, al.user_id, al.action, al.resource_type, al.resource_id, al.changes, al.created_at
      FROM audit_logs al
      JOIN teams t ON al.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE t.owner_id = $1 OR tm.user_id = $1
    `
    const values: any[] = [userId]
    let paramCount = 2

    if (action) {
      query += ` AND al.action = $${paramCount}`
      values.push(action)
      paramCount++
    }

    query += ` ORDER BY al.created_at DESC LIMIT $${paramCount} OFFSET $${paramCount + 1}`
    values.push(limit, offset)

    const result = await pool.query(query, values)

    res.json({
      success: true,
      data: result.rows
    })
  } catch (err) {
    console.error('Error fetching audit logs:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch audit logs'
    })
  }
})

// Get audit log by ID
router.get('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const logId = req.params.id

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    const result = await pool.query(`
      SELECT al.id, al.team_id, al.user_id, al.action, al.resource_type, al.resource_id, al.changes, al.created_at
      FROM audit_logs al
      JOIN teams t ON al.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE al.id = $1 AND (t.owner_id = $2 OR tm.user_id = $2)
    `, [logId, userId])

    if (result.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Audit log not found' })
    }

    res.json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error fetching audit log:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch audit log'
    })
  }
})

// Create audit log (internal use only - typically called from other routes)
router.post('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const { team_id, action, resource_type, resource_id, changes } = req.body

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    if (!team_id || !action || !resource_type) {
      return res.status(400).json({ success: false, error: 'team_id, action, and resource_type are required' })
    }

    // Check if user has access to team
    const teamCheck = await pool.query(`
      SELECT id FROM teams t
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE t.id = $1 AND (t.owner_id = $2 OR tm.user_id = $2)
    `, [team_id, userId])

    if (teamCheck.rows.length === 0) {
      return res.status(403).json({ success: false, error: 'Forbidden' })
    }

    const result = await pool.query(`
      INSERT INTO audit_logs (team_id, user_id, action, resource_type, resource_id, changes)
      VALUES ($1, $2, $3, $4, $5, $6)
      RETURNING id, team_id, user_id, action, resource_type, resource_id, changes, created_at
    `, [team_id, userId, action, resource_type, resource_id || null, changes ? JSON.stringify(changes) : null])

    res.status(201).json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error creating audit log:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to create audit log'
    })
  }
})

export default router
