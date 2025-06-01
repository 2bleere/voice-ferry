#!/usr/bin/env node

const express = require('express');
const redis = require('redis');
const app = express();

app.use(express.json());

// Create Redis client
const redisClient = redis.createClient({
  url: 'redis://localhost:6379'
});

// Connect to Redis
redisClient.connect().catch(console.error);

// Test endpoint
app.get('/api/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Session limits endpoints for testing
app.get('/api/sessions/limits/:username', async (req, res) => {
  try {
    const { username } = req.params;
    const limit = await redisClient.hGet('session_limits', username);
    
    if (limit) {
      res.json({ username, limit: parseInt(limit) });
    } else {
      // Return default limit
      res.json({ username, limit: 3, isDefault: true });
    }
  } catch (error) {
    console.error('Error getting user limit:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

app.put('/api/sessions/limits/:username', async (req, res) => {
  try {
    const { username } = req.params;
    const { limit } = req.body;
    
    if (typeof limit !== 'number' || limit < 0) {
      return res.status(400).json({ error: 'Invalid limit value' });
    }
    
    await redisClient.hSet('session_limits', username, limit.toString());
    res.json({ username, limit, message: 'Limit set successfully' });
  } catch (error) {
    console.error('Error setting user limit:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

app.delete('/api/sessions/limits/:username', async (req, res) => {
  try {
    const { username } = req.params;
    const deleted = await redisClient.hDel('session_limits', username);
    
    if (deleted) {
      res.json({ username, message: 'Limit removed successfully' });
    } else {
      res.status(404).json({ error: 'User limit not found' });
    }
  } catch (error) {
    console.error('Error deleting user limit:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Session stats endpoint
app.get('/api/sessions/stats', async (req, res) => {
  try {
    const stats = {
      total_sessions: 0,
      active_sessions: 0,
      session_limits_enabled: true
    };
    res.json(stats);
  } catch (error) {
    console.error('Error getting session stats:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Active sessions endpoint
app.get('/api/sessions/active', async (req, res) => {
  try {
    const sessions = [];
    res.json({ sessions, count: sessions.length });
  } catch (error) {
    console.error('Error getting active sessions:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Test server listening on port ${PORT}`);
  console.log('Per-user session limits API endpoints are ready for testing');
});

// Graceful shutdown
process.on('SIGINT', async () => {
  console.log('Shutting down test server...');
  await redisClient.quit();
  process.exit(0);
});
