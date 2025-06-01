const express = require('express');
const router = express.Router();

// Get B2BUA service status
router.get('/status', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const status = await monitoringService.getB2BUAStatus();
    res.json(status);
  } catch (error) {
    console.error('Get status error:', error);
    res.status(500).json({ error: 'Failed to get B2BUA status' });
  }
});

// Restart B2BUA service
router.post('/restart', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    
    const result = await monitoringService.restartB2BUA();
    res.json(result);
  } catch (error) {
    console.error('Restart error:', error);
    res.status(500).json({ error: 'Failed to restart B2BUA service' });
  }
});

// Get routing rules
router.get('/routing/rules', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    
    const rules = await configManager.getRoutingRules();
    res.json(rules);
  } catch (error) {
    console.error('Get routing rules error:', error);
    res.status(500).json({ error: 'Failed to get routing rules' });
  }
});

// Add routing rule
router.post('/routing/rules', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const rule = req.body;
    
    const result = await configManager.addRoutingRule(rule);
    res.json(result);
  } catch (error) {
    console.error('Add routing rule error:', error);
    res.status(500).json({ error: 'Failed to add routing rule' });
  }
});

// Update routing rule
router.put('/routing/rules/:ruleId', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { ruleId } = req.params;
    const rule = req.body;
    
    const result = await configManager.updateRoutingRule(ruleId, rule);
    res.json(result);
  } catch (error) {
    console.error('Update routing rule error:', error);
    res.status(500).json({ error: 'Failed to update routing rule' });
  }
});

// Delete routing rule
router.delete('/routing/rules/:ruleId', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { ruleId } = req.params;
    
    const result = await configManager.deleteRoutingRule(ruleId);
    res.json(result);
  } catch (error) {
    console.error('Delete routing rule error:', error);
    res.status(500).json({ error: 'Failed to delete routing rule' });
  }
});

// Test routing rule
router.post('/routing/test', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { fromUri, toUri, headers } = req.body;
    
    const result = await configManager.testRouting(fromUri, toUri, headers);
    res.json(result);
  } catch (error) {
    console.error('Test routing error:', error);
    res.status(500).json({ error: 'Failed to test routing' });
  }
});

// Get logs
router.get('/logs', async (req, res) => {
  try {
    const { monitoringService } = req.app.locals;
    const level = req.query.level || 'info';
    const lines = parseInt(req.query.lines) || 100;
    const follow = req.query.follow === 'true';
    
    if (follow) {
      // Set up streaming for log following
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive'
      });
      
      const stream = await monitoringService.streamLogs(level);
      stream.on('data', (data) => {
        res.write(`data: ${JSON.stringify(data)}\n\n`);
      });
      
      req.on('close', () => {
        stream.destroy();
      });
    } else {
      const logs = await monitoringService.getLogs(level, lines);
      res.json(logs);
    }
  } catch (error) {
    console.error('Get logs error:', error);
    res.status(500).json({ error: 'Failed to get logs' });
  }
});

module.exports = router;
