import { Router, Request, Response } from 'express'
import { AuthRequest } from '../middleware/auth'
import { pool } from '../index'

const router = Router()

// Get all alerts for user's teams
router.get('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const { severity, status } = req.query

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    let query = `
      SELECT a.id, a.team_id, a.title, a.severity, a.status, a.created_at, a.updated_at
      FROM alerts a
      JOIN teams t ON a.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE t.owner_id = $1 OR tm.user_id = $1
    `
    const values: any[] = [userId]
    let paramCount = 2

    if (severity) {
      query += ` AND a.severity = $${paramCount}`
      values.push(severity)
      paramCount++
    }

    if (status) {
      query += ` AND a.status = $${paramCount}`
      values.push(status)
      paramCount++
    }

    query += ' ORDER BY a.created_at DESC'

    const result = await pool.query(query, values)

    res.json({
      success: true,
      data: result.rows
    })
  } catch (err) {
    console.error('Error fetching alerts:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch alerts'
    })
  }
})

// Get alert by ID
router.get('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const alertId = req.params.id

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    const result = await pool.query(`
      SELECT a.id, a.team_id, a.title, a.severity, a.status, a.created_at, a.updated_at
      FROM alerts a
      JOIN teams t ON a.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE a.id = $1 AND (t.owner_id = $2 OR tm.user_id = $2)
    `, [alertId, userId])

    if (result.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Alert not found' })
    }

    res.json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error fetching alert:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch alert'
    })
  }
})

// Create alert
router.post('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const { team_id, title, severity = 'medium', status = 'open' } = req.body

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    if (!team_id || !title) {
      return res.status(400).json({ success: false, error: 'team_id and title are required' })
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
      INSERT INTO alerts (team_id, title, severity, status)
      VALUES ($1, $2, $3, $4)
      RETURNING id, team_id, title, severity, status, created_at, updated_at
    `, [team_id, title, severity, status])

    res.status(201).json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error creating alert:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to create alert'
    })
  }
})

// Update alert
router.patch('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const alertId = req.params.id
    const { title, severity, status } = req.body

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    // Check access
    const alertCheck = await pool.query(`
      SELECT a.id FROM alerts a
      JOIN teams t ON a.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE a.id = $1 AND (t.owner_id = $2 OR tm.user_id = $2)
    `, [alertId, userId])

    if (alertCheck.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Alert not found' })
    }

    const updates = []
    const values = []
    let paramCount = 1

    if (title) { updates.push(`title = $${paramCount++}`); values.push(title) }
    if (severity) { updates.push(`severity = $${paramCount++}`); values.push(severity) }
    if (status) { updates.push(`status = $${paramCount++}`); values.push(status) }
    updates.push(`updated_at = NOW()`)

    if (updates.length === 1) {
      return res.status(400).json({ success: false, error: 'No updates provided' })
    }

    values.push(alertId)
    const result = await pool.query(`
      UPDATE alerts
      SET ${updates.join(', ')}
      WHERE id = $${paramCount}
      RETURNING id, team_id, title, severity, status, created_at, updated_at
    `, values)

    res.json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error updating alert:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to update alert'
    })
  }
})

export default router
