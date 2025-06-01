const express = require('express');
const router = express.Router();

// Get session limits configuration
router.get('/limits', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    
    const limits = await configManager.getSessionLimitsConfig();
    res.json(limits);
  } catch (error) {
    console.error('Get session limits error:', error);
    res.status(500).json({ error: 'Failed to get session limits configuration' });
  }
});

// Update session limits configuration
router.put('/limits', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const limits = req.body;
    
    const result = await configManager.updateSessionLimitsConfig(limits);
    res.json(result);
  } catch (error) {
    console.error('Update session limits error:', error);
    res.status(500).json({ error: 'Failed to update session limits configuration' });
  }
});

// Get user-specific session limit
router.get('/limits/:username', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { username } = req.params;
    
    const limit = await configManager.getUserSessionLimit(username);
    res.json({ username, limit });
  } catch (error) {
    console.error('Get user session limit error:', error);
    res.status(500).json({ error: 'Failed to get user session limit' });
  }
});

// Set user-specific session limit
router.put('/limits/:username', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { username } = req.params;
    const { limit } = req.body;
    
    if (limit === undefined) {
      return res.status(400).json({ error: 'Limit value is required' });
    }
    
    const result = await configManager.setUserSessionLimit(username, parseInt(limit, 10));
    res.json(result);
  } catch (error) {
    console.error('Set user session limit error:', error);
    res.status(500).json({ error: 'Failed to set user session limit' });
  }
});

// Delete user-specific session limit (revert to default)
router.delete('/limits/:username', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { username } = req.params;
    
    const result = await configManager.deleteUserSessionLimit(username);
    res.json(result);
  } catch (error) {
    console.error('Delete user session limit error:', error);
    res.status(500).json({ error: 'Failed to delete user session limit' });
  }
});

// Get active sessions
router.get('/active', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const username = req.query.username;
    
    const sessions = await monitoringService.getActiveSessions(username);
    res.json(sessions);
  } catch (error) {
    console.error('Get active sessions error:', error);
    res.status(500).json({ error: 'Failed to get active sessions' });
  }
});

// Get session statistics
router.get('/stats', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const timeRange = req.query.timeRange || '1h';
    
    const stats = await monitoringService.getSessionStatistics(timeRange);
    res.json(stats);
  } catch (error) {
    console.error('Get session stats error:', error);
    res.status(500).json({ error: 'Failed to get session statistics' });
  }
});

// Get user session overview
router.get('/users', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const page = parseInt(req.query.page) || 1;
    const limit = parseInt(req.query.limit) || 50;
    const sort = req.query.sort || 'sessions_desc';
    
    const users = await monitoringService.getUserSessionOverview(page, limit, sort);
    res.json(users);
  } catch (error) {
    console.error('Get user session overview error:', error);
    res.status(500).json({ error: 'Failed to get user session overview' });
  }
});

// Get specific user sessions
router.get('/users/:username', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const { username } = req.params;
    
    const userSessions = await monitoringService.getUserSessions(username);
    res.json(userSessions);
  } catch (error) {
    console.error('Get user sessions error:', error);
    res.status(500).json({ error: 'Failed to get user sessions' });
  }
});

// Terminate specific session
router.delete('/:sessionId', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const { sessionId } = req.params;
    const reason = req.body.reason || 'Terminated by administrator';
    
    const result = await monitoringService.terminateSession(sessionId, reason);
    res.json(result);
  } catch (error) {
    console.error('Terminate session error:', error);
    res.status(500).json({ error: 'Failed to terminate session' });
  }
});

// Terminate all user sessions
router.delete('/users/:username/sessions', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const { username } = req.params;
    const reason = req.body.reason || 'All sessions terminated by administrator';
    
    const result = await monitoringService.terminateUserSessions(username, reason);
    res.json(result);
  } catch (error) {
    console.error('Terminate user sessions error:', error);
    res.status(500).json({ error: 'Failed to terminate user sessions' });
  }
});

// Get session limit violations
router.get('/violations', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const timeRange = req.query.timeRange || '24h';
    const page = parseInt(req.query.page) || 1;
    const limit = parseInt(req.query.limit) || 50;
    
    const violations = await monitoringService.getSessionLimitViolations(timeRange, page, limit);
    res.json(violations);
  } catch (error) {
    console.error('Get session violations error:', error);
    res.status(500).json({ error: 'Failed to get session limit violations' });
  }
});

// Set temporary session limit for user
router.post('/users/:username/temporary-limit', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const { username } = req.params;
    const { limit, duration, reason } = req.body;
    
    if (!limit || !duration) {
      return res.status(400).json({ error: 'Limit and duration are required' });
    }
    
    const result = await monitoringService.setTemporarySessionLimit(username, limit, duration, reason);
    res.json(result);
  } catch (error) {
    console.error('Set temporary limit error:', error);
    res.status(500).json({ error: 'Failed to set temporary session limit' });
  }
});

// Get session history
router.get('/history', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const username = req.query.username;
    const timeRange = req.query.timeRange || '24h';
    const page = parseInt(req.query.page) || 1;
    const limit = parseInt(req.query.limit) || 100;
    
    const history = await monitoringService.getSessionHistory(username, timeRange, page, limit);
    res.json(history);
  } catch (error) {
    console.error('Get session history error:', error);
    res.status(500).json({ error: 'Failed to get session history' });
  }
});

module.exports = router;
