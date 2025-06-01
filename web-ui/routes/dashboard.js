const express = require('express');
const router = express.Router();

// Get dashboard overview
router.get('/overview', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const overview = await monitoringService.getDashboardOverview();
    res.json(overview);
  } catch (error) {
    console.error('Dashboard overview error:', error);
    res.status(500).json({ error: 'Failed to get dashboard overview' });
  }
});

// Get system status
router.get('/status', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const status = await monitoringService.getSystemStatus();
    res.json(status);
  } catch (error) {
    console.error('System status error:', error);
    res.status(500).json({ error: 'Failed to get system status' });
  }
});

// Get call statistics
router.get('/call-stats', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const timeRange = req.query.timeRange || '1h';
    
    const stats = await monitoringService.getCallStatistics(timeRange);
    res.json(stats);
  } catch (error) {
    console.error('Call stats error:', error);
    res.status(500).json({ error: 'Failed to get call statistics' });
  }
});

// Get active calls
router.get('/active-calls', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const activeCalls = await monitoringService.getActiveCalls();
    res.json(activeCalls);
  } catch (error) {
    console.error('Active calls error:', error);
    res.status(500).json({ error: 'Failed to get active calls' });
  }
});

// Get session limits overview
router.get('/session-limits', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const sessionLimits = await monitoringService.getSessionLimitsOverview();
    res.json(sessionLimits);
  } catch (error) {
    console.error('Session limits error:', error);
    res.status(500).json({ error: 'Failed to get session limits data' });
  }
});

// Get performance metrics
router.get('/performance', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const timeRange = req.query.timeRange || '1h';
    
    const performance = await monitoringService.getPerformanceMetrics(timeRange);
    res.json(performance);
  } catch (error) {
    console.error('Performance metrics error:', error);
    res.status(500).json({ error: 'Failed to get performance metrics' });
  }
});

// Get recent events/alerts
router.get('/events', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const limit = parseInt(req.query.limit) || 50;
    const severity = req.query.severity || 'all';
    
    const events = await monitoringService.getRecentEvents(limit, severity);
    res.json(events);
  } catch (error) {
    console.error('Events error:', error);
    res.status(500).json({ error: 'Failed to get recent events' });
  }
});

module.exports = router;
