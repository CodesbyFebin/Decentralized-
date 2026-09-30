import { Router, Request, Response } from 'express'
import { AuthRequest } from '../middleware/auth'
import { pool } from '../index'

const router = Router()

// Get all nodes for user's teams
router.get('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    const result = await pool.query(`
      SELECT n.id, n.team_id, n.name, n.status, n.cpu_cores, n.memory_gb, n.created_at
      FROM nodes n
      JOIN teams t ON n.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE t.owner_id = $1 OR tm.user_id = $1
      ORDER BY n.created_at DESC
    `, [userId])

    res.json({
      success: true,
      data: result.rows
    })
  } catch (err) {
    console.error('Error fetching nodes:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch nodes'
    })
  }
})

// Get node by ID
router.get('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const nodeId = req.params.id
    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    const result = await pool.query(`
      SELECT n.id, n.team_id, n.name, n.status, n.cpu_cores, n.memory_gb, n.created_at
      FROM nodes n
      JOIN teams t ON n.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE n.id = $1 AND (t.owner_id = $2 OR tm.user_id = $2)
    `, [nodeId, userId])

    if (result.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Node not found' })
    }

    res.json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error fetching node:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to fetch node'
    })
  }
})

// Create node
router.post('/', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const { team_id, name, cpu_cores = 4, memory_gb = 8 } = req.body

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    if (!team_id || !name) {
      return res.status(400).json({ success: false, error: 'team_id and name are required' })
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
      INSERT INTO nodes (team_id, name, status, cpu_cores, memory_gb)
      VALUES ($1, $2, 'active', $3, $4)
      RETURNING id, team_id, name, status, cpu_cores, memory_gb, created_at
    `, [team_id, name, cpu_cores, memory_gb])

    res.status(201).json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error creating node:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to create node'
    })
  }
})

// Update node
router.patch('/:id', async (req: AuthRequest, res: Response) => {
  try {
    const userId = req.user?.id
    const nodeId = req.params.id
    const { name, status, cpu_cores, memory_gb } = req.body

    if (!userId) {
      return res.status(401).json({ success: false, error: 'Unauthorized' })
    }

    // Check access
    const nodeCheck = await pool.query(`
      SELECT n.id, n.team_id FROM nodes n
      JOIN teams t ON n.team_id = t.id
      LEFT JOIN team_members tm ON t.id = tm.team_id
      WHERE n.id = $1 AND (t.owner_id = $2 OR tm.user_id = $2)
    `, [nodeId, userId])

    if (nodeCheck.rows.length === 0) {
      return res.status(404).json({ success: false, error: 'Node not found' })
    }

    const updates = []
    const values = []
    let paramCount = 1

    if (name) { updates.push(`name = $${paramCount++}`); values.push(name) }
    if (status) { updates.push(`status = $${paramCount++}`); values.push(status) }
    if (cpu_cores) { updates.push(`cpu_cores = $${paramCount++}`); values.push(cpu_cores) }
    if (memory_gb) { updates.push(`memory_gb = $${paramCount++}`); values.push(memory_gb) }

    if (updates.length === 0) {
      return res.status(400).json({ success: false, error: 'No updates provided' })
    }

    values.push(nodeId)
    const result = await pool.query(`
      UPDATE nodes
      SET ${updates.join(', ')}
      WHERE id = $${paramCount}
      RETURNING id, team_id, name, status, cpu_cores, memory_gb, created_at
    `, values)

    res.json({
      success: true,
      data: result.rows[0]
    })
  } catch (err) {
    console.error('Error updating node:', err)
    res.status(500).json({
      success: false,
      error: 'Failed to update node'
    })
  }
})

export default router
