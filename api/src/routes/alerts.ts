import { Router, Request, Response } from 'express'
import { AuthRequest } from '../middleware/auth'
import { pool } from '../index'

const router = Router()

// Get all
router.get('/', async (req: AuthRequest, res: Response) => {
  try {
    res.json({
      success: true,
      data: []
    })
  } catch (err) {
    res.status(500).json({
      success: false,
      error: 'Failed to fetch data'
    })
  }
})

// Get by ID
router.get('/:id', async (req: AuthRequest, res: Response) => {
  try {
    res.json({
      success: true,
      data: {}
    })
  } catch (err) {
    res.status(500).json({
      success: false,
      error: 'Failed to fetch data'
    })
  }
})

// Create
router.post('/', async (req: AuthRequest, res: Response) => {
  try {
    res.status(201).json({
      success: true,
      data: {}
    })
  } catch (err) {
    res.status(500).json({
      success: false,
      error: 'Failed to create'
    })
  }
})

export default router
