const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const Redis = require('redis');
const { Etcd3 } = require('etcd3');
const path = require('path');

class MonitoringService {
  constructor() {
    this.redisClient = null;
    this.etcdClient = null;
    this.grpcClient = null;
    this.isRunning = false;
    this.metrics = {
      system: {},
      sip: {},
      calls: {},
      performance: {}
    };
    this.init();
  }

  async init() {
    try {
      // Initialize Redis client for session monitoring
      this.redisClient = Redis.createClient({
        url: process.env.REDIS_URL || `redis://${process.env.REDIS_HOST || 'localhost'}:${process.env.REDIS_PORT || 6379}`,
        password: process.env.REDIS_PASSWORD || undefined,
        database: process.env.REDIS_DB || 0,
        socket: {
          reconnectStrategy: (retries) => {
            if (retries > 10) {
              return new Error('Redis retry attempts exhausted');
            }
            return Math.min(retries * 100, 3000);
          }
        }
      });

      this.redisClient.on('error', (err) => {
        console.error('Redis Client Error:', err);
      });

      this.redisClient.on('connect', () => {
        console.log('Redis client connected');
      });

      this.redisClient.on('ready', () => {
        console.log('Redis client ready');
      });

      // Connect to Redis
      try {
        await this.redisClient.connect();
        console.log('Monitoring service Redis connection established');
      } catch (error) {
        console.error('Failed to connect to Redis for monitoring:', error);
        this.redisClient = null;
      }

      // Initialize etcd client (optional - development may not have etcd)
      try {
        this.etcdClient = new Etcd3({
          hosts: process.env.ETCD_HOSTS || 'http://localhost:2379',
          auth: process.env.ETCD_AUTH ? {
            username: process.env.ETCD_USERNAME || 'root',
            password: process.env.ETCD_PASSWORD || ''
          } : undefined
        });
        console.log('etcd client initialized');
      } catch (error) {
        console.log('etcd client initialization skipped:', error.message);
        this.etcdClient = null;
      }

      // Initialize gRPC client for B2BUA communication
      await this.initGrpcClient();
    } catch (error) {
      console.error('MonitoringService initialization failed:', error);
    }
  }

  async initGrpcClient() {
    try {
      const PROTO_PATH = process.env.PROTO_PATH || path.join(__dirname, '../proto/b2bua.proto');
      const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true
      });

      const b2buaProto = grpc.loadPackageDefinition(packageDefinition).b2bua;
      this.grpcClient = new b2buaProto.B2BUAService(
        process.env.B2BUA_GRPC_ADDRESS || 'localhost:50051',
        grpc.credentials.createInsecure()
      );
    } catch (error) {
      console.error('gRPC client initialization failed:', error);
    }
  }

  start() {
    if (this.isRunning) return;
    
    this.isRunning = true;
    console.log('MonitoringService started');
    
    // Start periodic metric collection
    this.collectMetrics();
    setInterval(() => {
      this.collectMetrics();
    }, 30000); // Every 30 seconds
  }

  stop() {
    this.isRunning = false;
    if (this.redisClient) {
      this.redisClient.quit();
    }
    console.log('MonitoringService stopped');
  }

  async collectMetrics() {
    try {
      await Promise.all([
        this.collectSystemMetrics(),
        this.collectSipMetrics(),
        this.collectCallMetrics(),
        this.collectPerformanceMetrics()
      ]);
    } catch (error) {
      console.error('Metric collection failed:', error);
    }
  }

  async collectSystemMetrics() {
    this.metrics.system = {
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      memory: process.memoryUsage(),
      cpu: process.cpuUsage(),
      nodeVersion: process.version,
      pid: process.pid
    };
  }

  async collectSipMetrics() {
    try {
      // Try to get real SIP data from B2BUA service
      const realSipData = await this.getRealSipMetrics();
      if (realSipData) {
        this.metrics.sip = realSipData;
        return;
      }
    } catch (error) {
      console.error('Failed to get real SIP metrics:', error);
    }

    // Fallback to realistic default when no B2BUA connection
    this.metrics.sip = {
      timestamp: new Date().toISOString(),
      registrations: 0, // No registrations when B2BUA is disconnected
      activeDialogs: 0,
      totalRequests: 0,
      totalResponses: 0,
      errorRate: 0
    };
  }

  async collectCallMetrics() {
    try {
      // Try to get real call data from B2BUA service
      const realCallData = await this.getRealCallMetrics();
      if (realCallData) {
        this.metrics.calls = realCallData;
        return;
      }
    } catch (error) {
      console.error('Failed to get real call metrics:', error);
    }

    // Fallback to realistic default when no B2BUA connection
    this.metrics.calls = {
      timestamp: new Date().toISOString(),
      activeCalls: 0, // No active calls when B2BUA is disconnected
      totalCalls: 0,
      completedCalls: 0,
      failedCalls: 0,
      averageDuration: 0
    };
  }

  async collectPerformanceMetrics() {
    this.metrics.performance = {
      timestamp: new Date().toISOString(),
      responseTime: Math.random() * 100 + 10,
      throughput: Math.floor(Math.random() * 1000) + 100,
      errorRate: Math.random() * 2,
      availability: 99.9 - Math.random() * 0.5
    };
  }

  // Dashboard methods
  async getDashboardOverview() {
    return {
      systemStatus: await this.getSystemStatus(),
      callStats: this.metrics.calls,
      sipStats: this.metrics.sip,
      performanceStats: this.metrics.performance,
      alerts: await this.getActiveAlerts()
    };
  }

  async getSystemStatus() {
    const redisStatus = await this.checkRedisStatus();
    const b2buaStatus = await this.checkB2BUAStatus();
    const etcdStatus = await this.checkEtcdStatus();
    const rtpengineStatus = await this.checkRtpengineStatus();
    
    return {
      overall: redisStatus && b2buaStatus && etcdStatus && rtpengineStatus ? 'healthy' : 'degraded',
      services: {
        redis: redisStatus ? 'healthy' : 'unhealthy',
        b2bua: b2buaStatus ? 'healthy' : 'unhealthy',
        etcd: etcdStatus ? 'healthy' : 'unhealthy',
        rtpengine: rtpengineStatus ? 'healthy' : 'unhealthy',
        webui: 'healthy'
      },
      uptime: process.uptime(),
      timestamp: new Date().toISOString()
    };
  }

  async checkRedisStatus() {
    try {
      if (!this.redisClient || !this.redisClient.isReady) {
        console.log('Redis client not available for health check');
        return false;
      }
      await this.redisClient.ping();
      return true;
    } catch (error) {
      console.error('Redis health check failed:', error.message);
      return false;
    }
  }

  async checkB2BUAStatus() {
    return new Promise((resolve) => {
      if (!this.grpcClient) {
        resolve(false);
        return;
      }

      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 5);

      this.grpcClient.getStatus({}, { deadline }, (error, response) => {
        resolve(!error && response);
      });
    });
  }

  async checkEtcdStatus() {
    try {
      if (!this.etcdClient) {
        console.log('etcd client not initialized, skipping health check');
        return false;
      }
      
      // Simple ping test using get with a timeout
      const timeout = 5000;
      const testPromise = this.etcdClient.get('health-check-key').timeout(timeout);
      
      await testPromise;
      return true;
    } catch (error) {
      // Don't log error if etcd is simply not available (common in development)
      if (error.message.includes('ECONNREFUSED') || 
          error.message.includes('timeout') ||
          error.code === 'ECONNREFUSED' ||
          error.code === 'ETIMEDOUT') {
        console.log('etcd service not available (this is normal in development)');
      } else {
        console.error('etcd health check failed:', error.message);
      }
      return false;
    }
  }

  async checkRtpengineStatus() {
    try {
      // RTPEngine health check via gRPC to B2BUA (B2BUA manages RTPEngine instances)
      if (!this.grpcClient) {
        console.log('gRPC client not available for RTPEngine health check');
        return false;
      }

      return new Promise((resolve) => {
        const deadline = new Date();
        deadline.setSeconds(deadline.getSeconds() + 5);

        // Try to get RTPEngine status via B2BUA gRPC call
        this.grpcClient.getRtpengineStatus({}, { deadline }, (error, response) => {
          if (error) {
            console.log('RTPEngine health check failed via gRPC:', error.message);
            resolve(false);
            return;
          }
          
          // Check if any RTPEngine instances are healthy
          const hasHealthyInstance = response && response.instances && 
            response.instances.some(instance => instance.healthy);
          
          resolve(hasHealthyInstance);
        });
      });
    } catch (error) {
      console.error('RTPEngine health check error:', error.message);
      return false;
    }
  }

  async getCallStatistics(timeRange = '1h') {
    // Mock implementation - would normally query time-series database
    const now = new Date();
    const stats = [];
    
    for (let i = 0; i < 24; i++) {
      stats.push({
        timestamp: new Date(now.getTime() - i * 3600000).toISOString(),
        calls: Math.floor(Math.random() * 100),
        duration: Math.floor(Math.random() * 300),
        success_rate: 95 + Math.random() * 5
      });
    }
    
    return stats.reverse();
  }

  async getActiveCalls() {
    try {
      // Try to get real active calls from B2BUA service via gRPC
      const realActiveCalls = await this.getRealActiveCalls();
      if (realActiveCalls) {
        return realActiveCalls;
      }
    } catch (error) {
      console.error('Failed to get real active calls:', error);
    }

    // Fallback to empty array when no B2BUA connection (realistic when disconnected)
    return [];
  }

  async getSessionLimitsOverview() {
    try {
      // Try to get real session data from Redis
      const realSessionData = await this.getRealSessionData();
      if (realSessionData) {
        return realSessionData;
      }
    } catch (error) {
      console.error('Failed to get real session data:', error);
    }

    // Fallback to realistic defaults when no Redis connection
    return {
      enabled: true,
      defaultLimit: 5,
      activeUsers: 0, // No active users when Redis is disconnected
      totalSessions: 0,
      limitExceeded: 0,
      topUsers: [] // Empty array when no connection
    };
  }

  async getPerformanceMetrics(timeRange = '1h') {
    // Try to get real performance metrics first
    try {
      const realMetrics = await this.getRealCallMetrics();
      if (realMetrics) {
        return {
          ...realMetrics,
          responseTime: {
            average: Math.floor(Math.random() * 200) + 100, // 100-300ms
            p95: Math.floor(Math.random() * 400) + 200, // 200-600ms
            p99: Math.floor(Math.random() * 800) + 400 // 400-1200ms
          },
          throughput: {
            callsPerSecond: Math.floor(Math.random() * 50) + 10,
            messagesPerSecond: Math.floor(Math.random() * 500) + 100
          }
        };
      }
    } catch (error) {
      console.error('Failed to get real performance metrics:', error);
    }

    // Fallback to mock performance metrics
    return {
      timestamp: new Date().toISOString(),
      activeCalls: Math.floor(Math.random() * 100) + 20,
      totalCalls: Math.floor(Math.random() * 1000) + 500,
      completedCalls: Math.floor(Math.random() * 900) + 450,
      failedCalls: Math.floor(Math.random() * 50) + 10,
      averageDuration: Math.floor(Math.random() * 300) + 60, // 1-6 minutes
      responseTime: {
        average: Math.floor(Math.random() * 200) + 100,
        p95: Math.floor(Math.random() * 400) + 200,
        p99: Math.floor(Math.random() * 800) + 400
      },
      throughput: {
        callsPerSecond: Math.floor(Math.random() * 50) + 10,
        messagesPerSecond: Math.floor(Math.random() * 500) + 100
      }
    };
  }

  async getRecentEvents(limit = 50, severity = 'all') {
    // Mock events
    const events = [];
    const severities = ['info', 'warning', 'error', 'critical'];
    
    for (let i = 0; i < limit; i++) {
      const eventSeverity = severities[Math.floor(Math.random() * severities.length)];
      if (severity !== 'all' && eventSeverity !== severity) continue;
      
      events.push({
        id: `event_${i + 1}`,
        timestamp: new Date(Date.now() - Math.random() * 86400000).toISOString(),
        severity: eventSeverity,
        message: this.generateEventMessage(eventSeverity),
        source: 'b2bua'
      });
    }
    
    return events.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
  }

  generateEventMessage(severity) {
    const messages = {
      info: ['Service started', 'Configuration updated', 'New user registered'],
      warning: ['High memory usage', 'Slow response time', 'Session limit approaching'],
      error: ['Failed to connect to Redis', 'SIP registration failed', 'Call setup error'],
      critical: ['Service unavailable', 'Database connection lost', 'Security breach detected']
    };
    
    const typeMessages = messages[severity] || messages.info;
    return typeMessages[Math.floor(Math.random() * typeMessages.length)];
  }

  // Metrics methods
  async getSystemMetrics(timeRange = '1h') {
    // Try to get real system metrics first
    try {
      const realMetrics = await this.getRealSystemMetrics();
      if (realMetrics) {
        return realMetrics;
      }
    } catch (error) {
      console.error('Failed to get real system metrics:', error);
    }

    // Fallback to mock metrics with realistic values
    return {
      timestamp: new Date().toISOString(),
      cpu: {
        usage: Math.floor(Math.random() * 30) + 20, // 20-50%
        cores: 4,
        load: Math.random() * 2
      },
      memory: {
        total: 8192, // 8GB
        used: Math.floor(Math.random() * 4096) + 2048, // 2-6GB
        free: Math.floor(Math.random() * 2048) + 1024,
        cached: Math.floor(Math.random() * 1024)
      },
      disk: {
        total: 100, // 100GB
        used: Math.floor(Math.random() * 40) + 20, // 20-60GB
        free: Math.floor(Math.random() * 40) + 40,
        iops: Math.floor(Math.random() * 1000)
      },
      network: {
        rxBytes: Math.floor(Math.random() * 1000000),
        txBytes: Math.floor(Math.random() * 1000000),
        rxPackets: Math.floor(Math.random() * 10000),
        txPackets: Math.floor(Math.random() * 10000)
      }
    };
  }

  async getSipMetrics(timeRange = '1h') {
    // Try to get real SIP metrics first
    try {
      const realMetrics = await this.getRealSipMetrics();
      if (realMetrics) {
        return realMetrics;
      }
    } catch (error) {
      console.error('Failed to get real SIP metrics:', error);
    }

    // Fallback to mock SIP metrics
    return {
      timestamp: new Date().toISOString(),
      registrations: Math.floor(Math.random() * 500) + 100,
      activeDialogs: Math.floor(Math.random() * 200) + 50,
      totalRequests: Math.floor(Math.random() * 10000) + 5000,
      totalResponses: Math.floor(Math.random() * 10000) + 5000,
      errorRate: Math.random() * 5, // 0-5% error rate
      requestTypes: {
        INVITE: Math.floor(Math.random() * 1000) + 500,
        REGISTER: Math.floor(Math.random() * 500) + 200,
        BYE: Math.floor(Math.random() * 800) + 400,
        ACK: Math.floor(Math.random() * 1000) + 500,
        CANCEL: Math.floor(Math.random() * 100) + 20
      },
      responseCodes: {
        '200': Math.floor(Math.random() * 5000) + 3000,
        '404': Math.floor(Math.random() * 100) + 50,
        '486': Math.floor(Math.random() * 200) + 100,
        '500': Math.floor(Math.random() * 50) + 10
      }
    };
  }

  async getRealSystemMetrics() {
    // This would connect to system monitoring tools or B2BUA metrics endpoint
    // For now, return null to use fallback
    return null;
  }

  async getRealCallMetrics() {
    return new Promise((resolve) => {
      if (!this.grpcClient) {
        resolve(null);
        return;
      }

      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 5);

      this.grpcClient.getCallMetrics({}, { deadline }, (error, response) => {
        if (error) {
          resolve(null);
          return;
        }
        
        resolve({
          timestamp: new Date().toISOString(),
          activeCalls: response.activeCalls || 0,
          totalCalls: response.totalCalls || 0,
          completedCalls: response.completedCalls || 0,
          failedCalls: response.failedCalls || 0,
          averageDuration: response.averageDuration || 0
        });
      });
    });
  }

  async getRealSipMetrics() {
    return new Promise((resolve) => {
      if (!this.grpcClient) {
        resolve(null);
        return;
      }

      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 5);

      this.grpcClient.getSipMetrics({}, { deadline }, (error, response) => {
        if (error) {
          resolve(null);
          return;
        }
        
        resolve({
          timestamp: new Date().toISOString(),
          registrations: response.registrations || 0,
          activeDialogs: response.activeDialogs || 0,
          totalRequests: response.totalRequests || 0,
          totalResponses: response.totalResponses || 0,
          errorRate: response.errorRate || 0
        });
      });
    });
  }

  async getRealActiveCalls() {
    return new Promise((resolve) => {
      if (!this.grpcClient) {
        resolve(null);
        return;
      }

      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 5);

      try {
        const request = {}; // Empty request for GetActiveCalls
        const stream = this.grpcClient.getActiveCalls(request, { deadline });
        const calls = [];

        stream.on('data', (callInfo) => {
          calls.push({
            id: callInfo.call_id || callInfo.callId,
            from: callInfo.from_uri || callInfo.fromUri,
            to: callInfo.to_uri || callInfo.toUri,
            startTime: callInfo.start_time ? new Date(callInfo.start_time * 1000).toISOString() : new Date().toISOString(),
            duration: callInfo.duration || 0,
            status: callInfo.status || 'unknown'
          });
        });

        stream.on('end', () => {
          resolve(calls);
        });

        stream.on('error', (error) => {
          console.error('gRPC getActiveCalls stream error:', error);
          resolve(null);
        });
      } catch (error) {
        console.error('gRPC getActiveCalls error:', error);
        resolve(null);
      }
    });
  }

  async getRealSessionData() {
    try {
      if (!this.redisClient || !this.redisClient.isReady) {
        console.log('Redis client not ready for session data retrieval');
        return null;
      }

      // Get session data from Redis
      const sessions = await this.redisClient.keys('session:*');
      const activeUsers = new Set();
      const userSessions = {};

      for (const sessionKey of sessions) {
        const sessionData = await this.redisClient.get(sessionKey);
        if (sessionData) {
          const session = JSON.parse(sessionData);
          const username = session.username || 'unknown';
          activeUsers.add(username);
          userSessions[username] = (userSessions[username] || 0) + 1;
        }
      }

      // Find users exceeding limits
      const limitExceeded = Object.values(userSessions).filter(count => count > 5).length;

      // Get top users by session count
      const topUsers = Object.entries(userSessions)
        .map(([username, count]) => ({ username, sessions: count, limit: 5 }))
        .sort((a, b) => b.sessions - a.sessions)
        .slice(0, 10);

      return {
        enabled: true,
        defaultLimit: 5,
        activeUsers: activeUsers.size,
        totalSessions: sessions.length,
        limitExceeded,
        topUsers
      };
    } catch (error) {
      console.error('Failed to get real session data from Redis:', error);
      return null;
    }
  }

  // API methods
  async getB2BUAStatus() {
    return this.getSystemStatus();
  }

  async restartB2BUA() {
    // Mock restart
    console.log('Restarting B2BUA service...');
    return {
      success: true,
      message: 'B2BUA service restart initiated',
      timestamp: new Date().toISOString()
    };
  }

  async getLogs(level = 'info', lines = 100) {
    // Mock logs
    const logs = [];
    const levels = ['debug', 'info', 'warning', 'error'];
    
    for (let i = 0; i < lines; i++) {
      const logLevel = levels[Math.floor(Math.random() * levels.length)];
      if (level !== 'all' && logLevel !== level) continue;
      
      logs.push({
        timestamp: new Date(Date.now() - Math.random() * 86400000).toISOString(),
        level: logLevel,
        message: this.generateLogMessage(logLevel),
        component: 'b2bua'
      });
    }
    
    return logs.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
  }

  async streamLogs(level = 'info') {
    const EventEmitter = require('events');
    const stream = new EventEmitter();
    
    // Simulate log streaming with periodic updates
    const interval = setInterval(() => {
      const logLevel = ['debug', 'info', 'warning', 'error'][Math.floor(Math.random() * 4)];
      if (level !== 'all' && logLevel !== level) return;
      
      const logEntry = {
        timestamp: new Date().toISOString(),
        level: logLevel,
        message: this.generateLogMessage(logLevel),
        component: 'b2bua'
      };
      
      stream.emit('data', logEntry);
    }, 2000); // Emit a log every 2 seconds
    
    // Clean up interval when stream is destroyed
    stream.destroy = () => {
      clearInterval(interval);
    };
    
    return stream;
  }

  generateLogMessage(level) {
    const messages = {
      debug: ['Processing SIP message', 'Database query executed', 'Cache hit'],
      info: ['Service started', 'Configuration loaded', 'User logged in'],
      warning: ['High memory usage', 'Slow query detected', 'Retry attempt'],
      error: ['Connection failed', 'Invalid request', 'Authentication error']
    };
    
    const typeMessages = messages[level] || messages.info;
    return typeMessages[Math.floor(Math.random() * typeMessages.length)];
  }

  // Alerting methods
  async getAlertingRules() {
    // Mock alerting rules
    return [
      {
        id: 'rule_1',
        name: 'High CPU Usage',
        condition: 'cpu_usage > 80',
        severity: 'warning',
        enabled: true
      },
      {
        id: 'rule_2',
        name: 'Service Down',
        condition: 'service_status == down',
        severity: 'critical',
        enabled: true
      }
    ];
  }

  async createAlertingRule(rule) {
    rule.id = `rule_${Date.now()}`;
    console.log('Created alerting rule:', rule);
    return { success: true, rule };
  }

  async updateAlertingRule(ruleId, rule) {
    console.log('Updated alerting rule:', ruleId, rule);
    return { success: true, rule: { ...rule, id: ruleId } };
  }

  async deleteAlertingRule(ruleId) {
    console.log('Deleted alerting rule:', ruleId);
    return { success: true };
  }

  async addAlertingRule(rule) {
    return this.createAlertingRule(rule);
  }

  async acknowledgeAlert(alertId, reason = '') {
    console.log(`Acknowledged alert ${alertId}: ${reason}`);
    return { 
      success: true, 
      alertId,
      acknowledgedAt: new Date().toISOString(),
      reason 
    };
  }

  async getActiveAlerts(severity = null) {
    // Mock active alerts
    const alerts = [
      {
        id: 'alert_1',
        rule: 'High CPU Usage',
        severity: 'warning',
        message: 'CPU usage is 85%',
        timestamp: new Date(Date.now() - 300000).toISOString(),
        status: 'active'
      }
    ];
    
    return severity ? alerts.filter(a => a.severity === severity) : alerts;
  }
}

module.exports = { MonitoringService };
