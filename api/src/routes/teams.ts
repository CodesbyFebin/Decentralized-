import { Router, Request, Response } from 'express'
import { AuthRequest } from '../middleware/auth'
import { pool } from '../index'

const router = Router()

// Get all teams for current user
router.get('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    const result = await pool.query(`
      SELECT t.id, t.name, t.owner_id, t.created_at
      FROM teams t
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE t.owner_id = $1 OR tm.user_id = $1
      GROUP BY t.id
      ORDER BY t.created_at DESC
    `, [userId])

    res.json({
      success: true,
      data: result.rows
    })
  } catch (err) {
    console.error('Error fetching teams:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch teams'
    })
  }
})

// Get team by ID
router.get('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const teamId = req.params.id
    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    const result = await pool.query(`
      SELECT t.id, t.name, t.owner_id, t.created_at
      FROM teams t
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE t.id = $1 AND (t.owner_id = $2 OR tm.user_id = $2)
    `, [teamId, userId])

    if (result.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Team not found' })
    }

    res.json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error fetching team:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch team'
    })
  }
})

// Create team
router.post('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const { name } = req.body

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    if (!name || typeof name !== 'string') {
      return res.status(400).json({ success: false, error: 'Team name is required' })
    }

    const result = await pool.query(`
      INSERT INTO teams (name, owner_id)
      VALUES ($1, $2)
      RETURNING id, name, owner_id, created_at
    `, [name, userId])

    res.status(201).json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error creating team:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to create team'
    })
  }
})

// Update team
router.patch('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const teamId = req.params.id
    const { name } = req.body

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    if (!name || typeof name !== 'string') {
      return res.status(400).json({ success: false, error: 'Team name is required' })
    }

    // Check if user is team owner
    const ownerCheck = await pool.query('SELECT owner_id FROM teams WHERE id = $1', [teamId])
    if (ownerCheck.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Team not found' })
    }

    if (ownerCheck.rows[0].owner_id !== userId) {
      return res.status(403).json({ success: false, error: 'Forbidden' })
    }

    const result = await pool.query(`
      UPDATE teams
      SET name = $1
      WHERE id = $2
      RETURNING id, name, owner_id, created_at
    `, [name, teamId])

    res.json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error updating team:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to update team'
    })
  }
})

// Delete team
router.delete('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const teamId = req.params.id

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    // Check if user is team owner
    const ownerCheck = await pool.query('SELECT owner_id FROM teams WHERE id = $1', [teamId])
    if (ownerCheck.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Team not found' })
    }

    if (ownerCheck.rows[0].owner_id !== userId) {
      return res.status(403).json({ success: false, error: 'Forbidden' })
    }

    await pool.query('DELETE FROM team_members WHERE team_id = $1', [teamId])
    await pool.query('DELETE FROM teams WHERE id = $1', [teamId])

    res.json({ success: true, message: 'Team deleted' })
  } catch (err) {
    console.error('Error deleting team:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to delete team'
    })
  }
})

export default router
