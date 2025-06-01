const fs = require('fs').promises;
const path = require('path');
const yaml = require('yaml');
const SessionLimitsService = require('./sessionLimitsService');

class ConfigManager {
  constructor() {
    this.configFile = process.env.CONFIG_FILE || path.join(__dirname, '../config/b2bua.yaml');
    this.backupDir = path.join(__dirname, '../data/backups');
    this.historyFile = path.join(__dirname, '../data/config-history.json');
    this.currentConfig = {};
    this.configHistory = [];
    this.loadConfig();
    this.sessionLimitsService = null;
  }

  async loadConfig() {
    try {
      const configData = await fs.readFile(this.configFile, 'utf8');
      this.currentConfig = yaml.parse(configData);
    } catch (error) {
      console.error('Failed to load config:', error);
      // Load default configuration
      this.currentConfig = this.getDefaultConfig();
      await this.saveConfig();
    }

    try {
      const historyData = await fs.readFile(this.historyFile, 'utf8');
      this.configHistory = JSON.parse(historyData);
    } catch (error) {
      this.configHistory = [];
    }
    
    // Initialize session limits service after config is loaded
    this.initSessionLimitsService();
  }

  initSessionLimitsService() {
    try {
      if (this.sessionLimitsService) {
        // Close existing service
        this.sessionLimitsService.close().catch(err => 
          console.error('Error closing previous session limits service:', err)
        );
      }
      
      this.sessionLimitsService = new SessionLimitsService(this.currentConfig);
      console.log('Session limits service initialized successfully');
    } catch (error) {
      console.error('Failed to initialize session limits service:', error);
      this.sessionLimitsService = null;
    }
  }

  // Session Limits Methods
  async getSessionLimitsConfig() {
    if (!this.sessionLimitsService) {
      throw new Error('Session limits service not available');
    }
    
    const config = this.currentConfig.redis || {};
    return {
      enabled: config.enable_session_limits || false,
      max_sessions_per_user: config.max_sessions_per_user || 3,
      session_limit_action: config.session_limit_action || 'reject',
      user_limits: await this.sessionLimitsService.getAllUserLimits()
    };
  }

  async updateSessionLimitsConfig(limits) {
    if (!this.sessionLimitsService) {
      throw new Error('Session limits service not available');
    }
    
    // Update the main configuration
    if (!this.currentConfig.redis) {
      this.currentConfig.redis = {};
    }
    
    this.currentConfig.redis.enable_session_limits = limits.enabled;
    this.currentConfig.redis.max_sessions_per_user = limits.max_sessions_per_user;
    this.currentConfig.redis.session_limit_action = limits.session_limit_action;
    
    await this.saveConfig();
    
    return {
      success: true,
      message: 'Session limits configuration updated successfully'
    };
  }

  async getUserSessionLimit(username) {
    if (!this.sessionLimitsService) {
      throw new Error('Session limits service not available');
    }
    
    return await this.sessionLimitsService.getUserLimit(username);
  }

  async setUserSessionLimit(username, limit) {
    if (!this.sessionLimitsService) {
      throw new Error('Session limits service not available');
    }
    
    const result = await this.sessionLimitsService.setUserLimit(username, limit);
    return {
      success: true,
      username,
      limit,
      message: `Session limit for user ${username} set to ${limit}`
    };
  }

  async deleteUserSessionLimit(username) {
    if (!this.sessionLimitsService) {
      throw new Error('Session limits service not available');
    }
    
    await this.sessionLimitsService.deleteUserLimit(username);
    return {
      success: true,
      username,
      message: `Session limit for user ${username} removed (will use default)`
    };
  }

  getDefaultConfig() {
    return {
      sip: {
        host: '0.0.0.0',
        port: 5060,
        transport: 'UDP',
        timeouts: {
          transaction: '32s',
          dialog: '12h',
          registration: '3600s'
        },
        tls: {
          enabled: false,
          cert_file: '',
          key_file: '',
          ca_file: ''
        }
      },
      redis: {
        enabled: true,
        host: 'localhost',
        port: 6379,
        password: '',
        database: 0,
        pool_size: 10,
        min_idle_conns: 0,
        max_idle_time: 300,
        conn_max_lifetime: 3600,
        timeout: 5,
        enable_session_limits: true,
        max_sessions_per_user: 5,
        session_limit_action: 'reject'
      },
      etcd: {
        enabled: true,
        endpoints: ['http://localhost:2379'],
        timeout: 5,
        username: '',
        password: '',
        prefix: '/voice-ferry-c4/',
        auto_sync_interval: 300,
        dial_timeout: 5,
        dial_keep_alive_time: 30,
        dial_keep_alive_timeout: 5
      },
      rtpengine: {
        enabled: true,
        host: 'localhost',
        port: 22222,
        timeout: 5,
        reconnect_interval: 10
      },
      logging: {
        level: 'info',
        format: 'json',
        output: 'stdout',
        file: {
          enabled: false,
          path: '/var/log/b2bua.log',
          max_size: '100MB',
          max_backups: 5
        }
      },
      metrics: {
        enabled: true,
        host: '0.0.0.0',
        port: 9090,
        path: '/metrics'
      },
      routing: {
        default_action: 'proxy',
        rules: []
      }
    };
  }

  async saveConfig() {
    try {
      await fs.mkdir(path.dirname(this.configFile), { recursive: true });
      const yamlData = yaml.stringify(this.currentConfig);
      await fs.writeFile(this.configFile, yamlData);

      // Add to history
      const historyEntry = {
        timestamp: new Date().toISOString(),
        config: JSON.parse(JSON.stringify(this.currentConfig)),
        action: 'save'
      };
      
      this.configHistory.unshift(historyEntry);
      if (this.configHistory.length > 50) {
        this.configHistory = this.configHistory.slice(0, 50);
      }

      await fs.mkdir(path.dirname(this.historyFile), { recursive: true });
      await fs.writeFile(this.historyFile, JSON.stringify(this.configHistory, null, 2));
    } catch (error) {
      console.error('Failed to save config:', error);
      throw error;
    }
  }

  async getCurrentConfig() {
    return JSON.parse(JSON.stringify(this.currentConfig));
  }

  async getConfigSection(section) {
    return this.currentConfig[section] || null;
  }

  async updateConfigSection(section, newConfig) {
    this.currentConfig[section] = newConfig;
    await this.saveConfig();
    return this.currentConfig[section];
  }

  async validateConfig(config) {
    const errors = [];
    const warnings = [];

    // SIP validation
    if (config.sip) {
      if (!config.sip.host) errors.push('SIP host is required');
      if (!config.sip.port || config.sip.port < 1 || config.sip.port > 65535) {
        errors.push('SIP port must be between 1 and 65535');
      }
      if (!['UDP', 'TCP', 'TLS', 'WS', 'WSS'].includes(config.sip.transport)) {
        errors.push('Invalid SIP transport');
      }
    }

    // Redis validation
    if (config.redis && config.redis.enabled) {
      if (!config.redis.host) errors.push('Redis host is required when enabled');
      if (!config.redis.port || config.redis.port < 1 || config.redis.port > 65535) {
        errors.push('Redis port must be between 1 and 65535');
      }
    }

    // etcd validation
    if (config.etcd && config.etcd.enabled) {
      if (!config.etcd.endpoints || !Array.isArray(config.etcd.endpoints) || config.etcd.endpoints.length === 0) {
        errors.push('etcd endpoints are required when enabled');
      }
      if (config.etcd.timeout && (config.etcd.timeout < 1 || config.etcd.timeout > 300)) {
        warnings.push('etcd timeout should be between 1 and 300 seconds');
      }
    }

    // RTPEngine validation
    if (config.rtpengine && config.rtpengine.enabled) {
      if (!config.rtpengine.host) errors.push('RTPEngine host is required when enabled');
      if (!config.rtpengine.port || config.rtpengine.port < 1 || config.rtpengine.port > 65535) {
        errors.push('RTPEngine port must be between 1 and 65535');
      }
    }

    return {
      valid: errors.length === 0,
      errors,
      warnings
    };
  }

  async applyConfig(config) {
    const validation = await this.validateConfig(config);
    if (!validation.valid) {
      return {
        success: false,
        error: 'Configuration validation failed',
        details: validation.errors
      };
    }

    this.currentConfig = config;
    await this.saveConfig();

    // In a real implementation, this would notify the B2BUA service to reload config
    console.log('Configuration applied successfully');

    return {
      success: true,
      message: 'Configuration applied successfully',
      warnings: validation.warnings
    };
  }

  async getConfigHistory(limit = 10) {
    return this.configHistory.slice(0, limit);
  }

  async createBackup(description = '') {
    try {
      await fs.mkdir(this.backupDir, { recursive: true });
      
      const backupId = `backup_${Date.now()}`;
      const backupFile = path.join(this.backupDir, `${backupId}.json`);
      
      const backup = {
        id: backupId,
        timestamp: new Date().toISOString(),
        description,
        config: JSON.parse(JSON.stringify(this.currentConfig))
      };

      await fs.writeFile(backupFile, JSON.stringify(backup, null, 2));

      return {
        success: true,
        backup: {
          id: backup.id,
          timestamp: backup.timestamp,
          description: backup.description
        }
      };
    } catch (error) {
      console.error('Backup creation failed:', error);
      return {
        success: false,
        error: 'Failed to create backup'
      };
    }
  }

  async restoreBackup(backupId) {
    try {
      const backupFile = path.join(this.backupDir, `${backupId}.json`);
      const backupData = await fs.readFile(backupFile, 'utf8');
      const backup = JSON.parse(backupData);

      const validation = await this.validateConfig(backup.config);
      if (!validation.valid) {
        return {
          success: false,
          error: 'Backup configuration is invalid',
          details: validation.errors
        };
      }

      this.currentConfig = backup.config;
      await this.saveConfig();

      return {
        success: true,
        message: 'Configuration restored from backup',
        backup: {
          id: backup.id,
          timestamp: backup.timestamp,
          description: backup.description
        }
      };
    } catch (error) {
      console.error('Backup restore failed:', error);
      return {
        success: false,
        error: 'Failed to restore backup'
      };
    }
  }

  async resetToDefaults(section = 'all') {
    try {
      const defaultConfig = this.getDefaultConfig();

      if (section === 'all') {
        this.currentConfig = defaultConfig;
      } else {
        this.currentConfig[section] = defaultConfig[section];
      }

      await this.saveConfig();

      return {
        success: true,
        message: `${section === 'all' ? 'All configuration' : section + ' configuration'} reset to defaults`
      };
    } catch (error) {
      console.error('Reset to defaults failed:', error);
      return {
        success: false,
        error: 'Failed to reset configuration'
      };
    }
  }

  // Routing methods
  async getRoutingRules() {
    return this.currentConfig.routing?.rules || [];
  }

  async addRoutingRule(rule) {
    if (!this.currentConfig.routing) {
      this.currentConfig.routing = { default_action: 'proxy', rules: [] };
    }
    
    rule.id = Date.now().toString();
    this.currentConfig.routing.rules.push(rule);
    await this.saveConfig();
    
    return { success: true, rule };
  }

  async updateRoutingRule(ruleId, updatedRule) {
    if (!this.currentConfig.routing?.rules) {
      return { success: false, error: 'No routing rules found' };
    }

    const index = this.currentConfig.routing.rules.findIndex(r => r.id === ruleId);
    if (index === -1) {
      return { success: false, error: 'Rule not found' };
    }

    this.currentConfig.routing.rules[index] = { ...updatedRule, id: ruleId };
    await this.saveConfig();

    return { success: true, rule: this.currentConfig.routing.rules[index] };
  }

  async deleteRoutingRule(ruleId) {
    if (!this.currentConfig.routing?.rules) {
      return { success: false, error: 'No routing rules found' };
    }

    const index = this.currentConfig.routing.rules.findIndex(r => r.id === ruleId);
    if (index === -1) {
      return { success: false, error: 'Rule not found' };
    }

    this.currentConfig.routing.rules.splice(index, 1);
    await this.saveConfig();

    return { success: true };
  }

  async testRouting(fromUri, toUri, headers = {}) {
    // Simple routing test implementation
    const rules = this.currentConfig.routing?.rules || [];
    
    for (const rule of rules) {
      if (this.matchesRule(fromUri, toUri, headers, rule)) {
        return {
          success: true,
          matched: true,
          rule: rule,
          action: rule.action || 'proxy'
        };
      }
    }

    return {
      success: true,
      matched: false,
      action: this.currentConfig.routing?.default_action || 'proxy'
    };
  }

  matchesRule(fromUri, toUri, headers, rule) {
    // Simple matching logic - in reality this would be more sophisticated
    if (rule.from_pattern && !fromUri.includes(rule.from_pattern)) {
      return false;
    }
    if (rule.to_pattern && !toUri.includes(rule.to_pattern)) {
      return false;
    }
    return true;
  }
}

module.exports = { ConfigManager };
