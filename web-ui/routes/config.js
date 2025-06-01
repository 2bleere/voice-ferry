const express = require('express');
const router = express.Router();
const Joi = require('joi');

// Configuration validation schemas
const sipConfigSchema = Joi.object({
  host: Joi.string().ip().required(),
  port: Joi.number().integer().min(1).max(65535).required(),
  transport: Joi.string().valid('UDP', 'TCP', 'TLS', 'WS', 'WSS').required(),
  timeouts: Joi.object({
    transaction: Joi.string().pattern(/^\d+[smh]$/).required(),
    dialog: Joi.string().pattern(/^\d+[smh]$/).required(),
    registration: Joi.string().pattern(/^\d+[smh]$/).required()
  }),
  tls: Joi.object({
    enabled: Joi.boolean(),
    cert_file: Joi.string().when('enabled', { is: true, then: Joi.required() }),
    key_file: Joi.string().when('enabled', { is: true, then: Joi.required() }),
    ca_file: Joi.string()
  })
});

const redisConfigSchema = Joi.object({
  enabled: Joi.boolean().required(),
  host: Joi.string().required(),
  port: Joi.number().integer().min(1).max(65535).required(),
  password: Joi.string().allow(''),
  database: Joi.number().integer().min(0).max(15).required(),
  pool_size: Joi.number().integer().min(1).required(),
  min_idle_conns: Joi.number().integer().min(0).required(),
  max_idle_time: Joi.number().integer().min(0).required(),
  conn_max_lifetime: Joi.number().integer().min(0).required(),
  timeout: Joi.number().integer().min(1).required(),
  enable_session_limits: Joi.boolean(),
  max_sessions_per_user: Joi.number().integer().min(1).when('enable_session_limits', { is: true, then: Joi.required() }),
  session_limit_action: Joi.string().valid('reject', 'terminate_oldest').when('enable_session_limits', { is: true, then: Joi.required() })
});

const etcdConfigSchema = Joi.object({
  enabled: Joi.boolean().required(),
  endpoints: Joi.array().items(Joi.string().uri()).min(1).required(),
  timeout: Joi.number().integer().min(1).max(300),
  username: Joi.string().allow(''),
  password: Joi.string().allow(''),
  prefix: Joi.string().required(),
  auto_sync_interval: Joi.number().integer().min(1),
  dial_timeout: Joi.number().integer().min(1),
  dial_keep_alive_time: Joi.number().integer().min(1),
  dial_keep_alive_timeout: Joi.number().integer().min(1)
});

const rtpengineConfigSchema = Joi.object({
  enabled: Joi.boolean().required(),
  host: Joi.string().required(),
  port: Joi.number().integer().min(1).max(65535).required(),
  timeout: Joi.number().integer().min(1),
  reconnect_interval: Joi.number().integer().min(1)
});

const loggingConfigSchema = Joi.object({
  level: Joi.string().valid('debug', 'info', 'warning', 'error').required(),
  format: Joi.string().valid('text', 'json').required(),
  output: Joi.string().valid('stdout', 'stderr', 'file').required(),
  file: Joi.object({
    enabled: Joi.boolean(),
    path: Joi.string().when('enabled', { is: true, then: Joi.required() }),
    max_size: Joi.string().pattern(/^\d+(MB|GB)$/),
    max_backups: Joi.number().integer().min(1)
  })
});

const metricsConfigSchema = Joi.object({
  enabled: Joi.boolean().required(),
  host: Joi.string().ip().required(),
  port: Joi.number().integer().min(1).max(65535).required(),
  path: Joi.string().required()
});

// Get current configuration
router.get('/', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    
    const config = await configManager.getCurrentConfig();
    res.json(config);
  } catch (error) {
    console.error('Get config error:', error);
    res.status(500).json({ error: 'Failed to get configuration' });
  }
});

// Get configuration section
router.get('/:section', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { section } = req.params;
    
    const sectionConfig = await configManager.getConfigSection(section);
    if (!sectionConfig) {
      return res.status(404).json({ error: 'Configuration section not found' });
    }
    
    res.json(sectionConfig);
  } catch (error) {
    console.error('Get config section error:', error);
    res.status(500).json({ error: 'Failed to get configuration section' });
  }
});

// Update configuration section
router.put('/:section', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { section } = req.params;
    const newConfig = req.body;
    
    // Validate configuration based on section
    let schema;
    switch (section) {
      case 'sip':
        schema = sipConfigSchema;
        break;
      case 'redis':
        schema = redisConfigSchema;
        break;
      case 'etcd':
        schema = etcdConfigSchema;
        break;
      case 'rtpengine':
        schema = rtpengineConfigSchema;
        break;
      case 'logging':
        schema = loggingConfigSchema;
        break;
      case 'metrics':
        schema = metricsConfigSchema;
        break;
      default:
        // For other sections, perform basic validation
        schema = Joi.object().unknown(true);
    }
    
    const { error, value } = schema.validate(newConfig);
    if (error) {
      return res.status(400).json({
        error: 'Validation failed',
        details: error.details.map(d => d.message)
      });
    }
    
    const result = await configManager.updateConfigSection(section, value);
    res.json({
      success: true,
      message: `${section} configuration updated successfully`,
      config: result
    });
  } catch (error) {
    console.error('Update config section error:', error);
    res.status(500).json({ error: 'Failed to update configuration section' });
  }
});

// Validate configuration
router.post('/validate', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const config = req.body;
    
    const validation = await configManager.validateConfig(config);
    res.json(validation);
  } catch (error) {
    console.error('Validate config error:', error);
    res.status(500).json({ error: 'Failed to validate configuration' });
  }
});

// Apply configuration
router.post('/apply', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const config = req.body;
    
    const result = await configManager.applyConfig(config);
    res.json(result);
  } catch (error) {
    console.error('Apply config error:', error);
    res.status(500).json({ error: 'Failed to apply configuration' });
  }
});

// Get configuration history
router.get('/history', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const limit = parseInt(req.query.limit) || 10;
    
    const history = await configManager.getConfigHistory(limit);
    res.json(history);
  } catch (error) {
    console.error('Get config history error:', error);
    res.status(500).json({ error: 'Failed to get configuration history' });
  }
});

// Backup configuration
router.post('/backup', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const description = req.body.description || 'Manual backup';
    
    const backup = await configManager.createBackup(description);
    res.json(backup);
  } catch (error) {
    console.error('Backup config error:', error);
    res.status(500).json({ error: 'Failed to create configuration backup' });
  }
});

// Restore configuration
router.post('/restore/:backupId', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const { backupId } = req.params;
    
    const result = await configManager.restoreBackup(backupId);
    res.json(result);
  } catch (error) {
    console.error('Restore config error:', error);
    res.status(500).json({ error: 'Failed to restore configuration backup' });
  }
});

// Reset to defaults
router.post('/reset', async (req, res) => {
  try {
    const { configManager } = req.app.locals;
    const section = req.body.section || 'all';
    
    const result = await configManager.resetToDefaults(section);
    res.json(result);
  } catch (error) {
    console.error('Reset config error:', error);
    res.status(500).json({ error: 'Failed to reset configuration' });
  }
});

module.exports = router;
