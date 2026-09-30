import { Router, Request, Response } from 'express'
import bcrypt from 'bcryptjs'
import jwt from 'jsonwebtoken'
import { v4 as uuidv4 } from 'uuid'
import { pool } from '../index'
import Joi from 'joi'

const router = Router()

// Validation schemas
const registerSchema = Joi.object({
  email: Joi.string().email().required(),
  password: Joi.string().min(8).required(),
  fullName: Joi.string().required()
})

const loginSchema = Joi.object({
  email: Joi.string().email().required(),
  password: Joi.string().required()
})

// Register endpoint
router.post('/register', async (req: Request, res: Response) => {
  try {
    const { error, value } = registerSchema.validate(req.body)
    if (error) {
      return res.status(400).json({
        success: false,
        error: error.details[0].message
      })
    }

    const { email, password, fullName } = value
    const client = await pool.connect()

    try {
      // Check if user exists
      const userCheck = await client.query('SELECT id FROM users WHERE email = $1', [email])
      if (userCheck.rows.length > 0) {
        return res.status(400).json({
          success: false,
          error: 'Email already registered'
        })
      }

      // Hash password
      const hashedPassword = await bcrypt.hash(password, 10)

      // Create user
      const userId = uuidv4()
      await client.query(
        'INSERT INTO users (id, email, password_hash, full_name) VALUES ($1, $2, $3, $4)',
        [userId, email, hashedPassword, fullName]
      )

      // Create JWT token
      const token = jwt.sign(
        { id: userId, email, role: 'user' },
        process.env.JWT_SECRET || 'your-secret-key',
        { expiresIn: '24h' }
      )

      res.status(201).json({
        success: true,
        data: {
          userId,
          email,
          token
        }
      })
    } finally {
      client.release()
    }
  } catch (err) {
    console.error(err)
    res.status(500).json({
      success: false,
      error: 'Registration failed'
    })
  }
})

// Login endpoint
router.post('/login', async (req: Request, res: Response) => {
  try {
    const { error, value } = loginSchema.validate(req.body)
    if (error) {
      return res.status(400).json({
        success: false,
        error: error.details[0].message
      })
    }

    const { email, password } = value
    const client = await pool.connect()

    try {
      // Find user
      const userResult = await client.query('SELECT * FROM users WHERE email = $1', [email])
      if (userResult.rows.length === 0) {
        return res.status(401).json({
          success: false,
          error: 'Invalid credentials'
        })
      }

      const user = userResult.rows[0]

      // Verify password
      const passwordMatch = await bcrypt.compare(password, user.password_hash)
      if (!passwordMatch) {
        return res.status(401).json({
          success: false,
          error: 'Invalid credentials'
        })
      }

      // Create JWT token
      const token = jwt.sign(
        { id: user.id, email: user.email, role: user.role },
        process.env.JWT_SECRET || 'your-secret-key',
        { expiresIn: '24h' }
      )

      res.json({
        success: true,
        data: {
          userId: user.id,
          email: user.email,
          fullName: user.full_name,
          role: user.role,
          token
        }
      })
    } finally {
      client.release()
    }
  } catch (err) {
    console.error(err)
    res.status(500).json({
      success: false,
      error: 'Login failed'
    })
  }
})

export default router
