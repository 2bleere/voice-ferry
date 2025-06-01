const express = require('express');
const router = express.Router();

// Get system metrics
router.get('/system', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const timeRange = req.query.timeRange || '1h';
    
    const metrics = await monitoringService.getSystemMetrics(timeRange);
    res.json(metrics);
  } catch (error) {
    console.error('Get system metrics error:', error);
    res.status(500).json({ error: 'Failed to get system metrics' });
  }
});

// Get SIP metrics
router.get('/sip', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const timeRange = req.query.timeRange || '1h';
    
    const metrics = await monitoringService.getSipMetrics(timeRange);
    res.json(metrics);
  } catch (error) {
    console.error('Get SIP metrics error:', error);
    res.status(500).json({ error: 'Failed to get SIP metrics' });
  }
});

// Get performance metrics
router.get('/performance', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const timeRange = req.query.timeRange || '1h';
    
    const metrics = await monitoringService.getPerformanceMetrics(timeRange);
    res.json(metrics);
  } catch (error) {
    console.error('Get performance metrics error:', error);
    res.status(500).json({ error: 'Failed to get performance metrics' });
  }
});

// Get Redis metrics
router.get('/redis', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const metrics = await monitoringService.getRedisMetrics();
    res.json(metrics);
  } catch (error) {
    console.error('Get Redis metrics error:', error);
    res.status(500).json({ error: 'Failed to get Redis metrics' });
  }
});

// Get etcd metrics
router.get('/etcd', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const metrics = await monitoringService.getEtcdMetrics();
    res.json(metrics);
  } catch (error) {
    console.error('Get etcd metrics error:', error);
    res.status(500).json({ error: 'Failed to get etcd metrics' });
  }
});

// Get rtpengine metrics
router.get('/rtpengine', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const metrics = await monitoringService.getRtpengineMetrics();
    res.json(metrics);
  } catch (error) {
    console.error('Get rtpengine metrics error:', error);
    res.status(500).json({ error: 'Failed to get rtpengine metrics' });
  }
});

// Get custom metrics
router.get('/custom', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const metricName = req.query.metric;
    const timeRange = req.query.timeRange || '1h';
    
    if (!metricName) {
      return res.status(400).json({ error: 'Metric name is required' });
    }
    
    const metrics = await monitoringService.getCustomMetric(metricName, timeRange);
    res.json(metrics);
  } catch (error) {
    console.error('Get custom metrics error:', error);
    res.status(500).json({ error: 'Failed to get custom metrics' });
  }
});

// Export metrics in Prometheus format
router.get('/prometheus', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const prometheusData = await monitoringService.getPrometheusMetrics();
    res.set('Content-Type', 'text/plain');
    res.send(prometheusData);
  } catch (error) {
    console.error('Get Prometheus metrics error:', error);
    res.status(500).json({ error: 'Failed to get Prometheus metrics' });
  }
});

// Get alerting rules
router.get('/alerts/rules', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const rules = await monitoringService.getAlertingRules();
    res.json(rules);
  } catch (error) {
    console.error('Get alerting rules error:', error);
    res.status(500).json({ error: 'Failed to get alerting rules' });
  }
});

// Add alerting rule
router.post('/alerts/rules', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const rule = req.body;
    
    const result = await monitoringService.addAlertingRule(rule);
    res.json(result);
  } catch (error) {
    console.error('Add alerting rule error:', error);
    res.status(500).json({ error: 'Failed to add alerting rule' });
  }
});

// Update alerting rule
router.put('/alerts/rules/:ruleId', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const { ruleId } = req.params;
    const rule = req.body;
    
    const result = await monitoringService.updateAlertingRule(ruleId, rule);
    res.json(result);
  } catch (error) {
    console.error('Update alerting rule error:', error);
    res.status(500).json({ error: 'Failed to update alerting rule' });
  }
});

// Delete alerting rule
router.delete('/alerts/rules/:ruleId', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const { ruleId } = req.params;
    
    const result = await monitoringService.deleteAlertingRule(ruleId);
    res.json(result);
  } catch (error) {
    console.error('Delete alerting rule error:', error);
    res.status(500).json({ error: 'Failed to delete alerting rule' });
  }
});

// Get active alerts
router.get('/alerts/active', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const severity = req.query.severity;
    
    const alerts = await monitoringService.getActiveAlerts(severity);
    res.json(alerts);
  } catch (error) {
    console.error('Get active alerts error:', error);
    res.status(500).json({ error: 'Failed to get active alerts' });
  }
});

// Acknowledge alert
router.post('/alerts/:alertId/acknowledge', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const { alertId } = req.params;
    const { reason } = req.body;
    
    const result = await monitoringService.acknowledgeAlert(alertId, reason);
    res.json(result);
  } catch (error) {
    console.error('Acknowledge alert error:', error);
    res.status(500).json({ error: 'Failed to acknowledge alert' });
  }
});

module.exports = router;
