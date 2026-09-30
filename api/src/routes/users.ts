import { Router, Request, Response } from 'express'
import { AuthRequest } from '../middleware/auth'
import { pool } from '../index'

const router = Router()

// Get current user
router.get('/me', async (req: AuthRequest, res: Response) => {
  try {
    const client = await pool.connect()
    try {
      const result = await client.query(
        'SELECT id, email, full_name, role FROM users WHERE id = $1',
        [req.user?.id]
      )
      if (result.rows.length === 0) {
        return res.status(404).json({
          success: false,
          error: 'User not found'
        })
      }
      res.json({
        success: true,
        data: result.rows[0]
      })
    } finally {
      client.release()
    }
  } catch (err) {
    res.status(500).json({
      success: false,
      error: 'Failed to fetch user'
    })
  }
})

// Get all users (admin only)
router.get('/', async (req: AuthRequest, res: Response) => {
  try {
    const client = await pool.connect()
    try {
      const result = await client.query(
        'SELECT id, email, full_name, role, created_at FROM users'
      )
      res.json({
        success: true,
        data: result.rows
      })
    } finally {
      client.release()
    }
  } catch (err) {
    res.status(500).json({
      success: false,
      error: 'Failed to fetch users'
    })
  }
})

export default router
