class WebSocketManager {
  constructor(wss, monitoringService) {
    this.wss = wss;
    this.monitoringService = monitoringService;
    this.clients = new Map();
    this.subscriptions = new Map();
    this.init();
  }

  init() {
    this.wss.on('connection', (ws, req) => {
      const clientId = this.generateClientId();
      this.clients.set(clientId, {
        ws,
        subscriptions: new Set(),
        lastActivity: new Date(),
        ipAddress: req.socket.remoteAddress
      });

      console.log(`WebSocket client connected: ${clientId}`);

      ws.on('message', (data) => {
        try {
          this.handleMessage(clientId, JSON.parse(data.toString()));
        } catch (error) {
          console.error('WebSocket message error:', error);
          ws.send(JSON.stringify({
            type: 'error',
            message: 'Invalid message format'
          }));
        }
      });

      ws.on('close', () => {
        this.handleDisconnect(clientId);
      });

      ws.on('error', (error) => {
        console.error(`WebSocket error for client ${clientId}:`, error);
        this.handleDisconnect(clientId);
      });

      // Send welcome message
      ws.send(JSON.stringify({
        type: 'connected',
        clientId,
        timestamp: new Date().toISOString()
      }));
    });

    // Start broadcasting updates
    this.startBroadcasting();
  }

  generateClientId() {
    return `client_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  handleMessage(clientId, message) {
    const client = this.clients.get(clientId);
    if (!client) return;

    client.lastActivity = new Date();

    switch (message.type) {
      case 'subscribe':
        this.handleSubscribe(clientId, message.channel);
        break;
      case 'unsubscribe':
        this.handleUnsubscribe(clientId, message.channel);
        break;
      case 'ping':
        client.ws.send(JSON.stringify({
          type: 'pong',
          timestamp: new Date().toISOString()
        }));
        break;
      default:
        client.ws.send(JSON.stringify({
          type: 'error',
          message: `Unknown message type: ${message.type}`
        }));
    }
  }

  handleSubscribe(clientId, channel) {
    const client = this.clients.get(clientId);
    if (!client) return;

    const validChannels = [
      'dashboard',
      'system-metrics',
      'call-stats',
      'active-calls',
      'session-limits',
      'alerts',
      'logs',
      'config-changes'
    ];

    if (!validChannels.includes(channel)) {
      client.ws.send(JSON.stringify({
        type: 'error',
        message: `Invalid channel: ${channel}`
      }));
      return;
    }

    client.subscriptions.add(channel);

    if (!this.subscriptions.has(channel)) {
      this.subscriptions.set(channel, new Set());
    }
    this.subscriptions.get(channel).add(clientId);

    client.ws.send(JSON.stringify({
      type: 'subscribed',
      channel,
      timestamp: new Date().toISOString()
    }));

    console.log(`Client ${clientId} subscribed to ${channel}`);
  }

  handleUnsubscribe(clientId, channel) {
    const client = this.clients.get(clientId);
    if (!client) return;

    client.subscriptions.delete(channel);

    if (this.subscriptions.has(channel)) {
      this.subscriptions.get(channel).delete(clientId);
      if (this.subscriptions.get(channel).size === 0) {
        this.subscriptions.delete(channel);
      }
    }

    client.ws.send(JSON.stringify({
      type: 'unsubscribed',
      channel,
      timestamp: new Date().toISOString()
    }));

    console.log(`Client ${clientId} unsubscribed from ${channel}`);
  }

  handleDisconnect(clientId) {
    const client = this.clients.get(clientId);
    if (!client) return;

    // Remove client from all subscriptions
    for (const channel of client.subscriptions) {
      if (this.subscriptions.has(channel)) {
        this.subscriptions.get(channel).delete(clientId);
        if (this.subscriptions.get(channel).size === 0) {
          this.subscriptions.delete(channel);
        }
      }
    }

    this.clients.delete(clientId);
    console.log(`WebSocket client disconnected: ${clientId}`);
  }

  startBroadcasting() {
    // Broadcast dashboard updates every 10 seconds
    setInterval(() => {
      this.broadcastDashboardUpdate();
    }, 10000);

    // Broadcast system metrics every 30 seconds
    setInterval(() => {
      this.broadcastSystemMetrics();
    }, 30000);

    // Broadcast call stats every 5 seconds
    setInterval(() => {
      this.broadcastCallStats();
    }, 5000);

    // Broadcast active calls every 15 seconds
    setInterval(() => {
      this.broadcastActiveCalls();
    }, 15000);

    // Broadcast session limits every 20 seconds
    setInterval(() => {
      this.broadcastSessionLimits();
    }, 20000);

    // Check for alerts every 30 seconds
    setInterval(() => {
      this.broadcastAlerts();
    }, 30000);
  }

  async broadcastDashboardUpdate() {
    try {
      const overview = await this.monitoringService.getDashboardOverview();
      this.broadcast('dashboard', {
        type: 'dashboard-update',
        data: overview,
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      console.error('Dashboard broadcast error:', error);
    }
  }

  async broadcastSystemMetrics() {
    try {
      const metrics = await this.monitoringService.getSystemMetrics();
      this.broadcast('system-metrics', {
        type: 'system-metrics',
        data: metrics,
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      console.error('System metrics broadcast error:', error);
    }
  }

  async broadcastCallStats() {
    try {
      const stats = await this.monitoringService.getCallStatistics('1h');
      this.broadcast('call-stats', {
        type: 'call-stats',
        data: stats,
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      console.error('Call stats broadcast error:', error);
    }
  }

  async broadcastActiveCalls() {
    try {
      const calls = await this.monitoringService.getActiveCalls();
      this.broadcast('active-calls', {
        type: 'active-calls',
        data: calls,
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      console.error('Active calls broadcast error:', error);
    }
  }

  async broadcastSessionLimits() {
    try {
      const limits = await this.monitoringService.getSessionLimitsOverview();
      this.broadcast('session-limits', {
        type: 'session-limits',
        data: limits,
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      console.error('Session limits broadcast error:', error);
    }
  }

  async broadcastAlerts() {
    try {
      const alerts = await this.monitoringService.getActiveAlerts();
      if (alerts.length > 0) {
        this.broadcast('alerts', {
          type: 'alerts',
          data: alerts,
          timestamp: new Date().toISOString()
        });
      }
    } catch (error) {
      console.error('Alerts broadcast error:', error);
    }
  }

  broadcast(channel, message) {
    const subscribers = this.subscriptions.get(channel);
    if (!subscribers || subscribers.size === 0) return;

    const messageStr = JSON.stringify(message);
    let sentCount = 0;
    let errorCount = 0;

    for (const clientId of subscribers) {
      const client = this.clients.get(clientId);
      if (!client || client.ws.readyState !== client.ws.OPEN) {
        // Remove dead client
        subscribers.delete(clientId);
        if (this.clients.has(clientId)) {
          this.handleDisconnect(clientId);
        }
        continue;
      }

      try {
        client.ws.send(messageStr);
        sentCount++;
      } catch (error) {
        console.error(`Failed to send message to client ${clientId}:`, error);
        errorCount++;
        // Remove problematic client
        this.handleDisconnect(clientId);
      }
    }

    if (sentCount > 0 || errorCount > 0) {
      console.log(`Broadcast to ${channel}: ${sentCount} sent, ${errorCount} errors`);
    }
  }

  // Public methods for manual broadcasting
  broadcastConfigChange(configSection, newConfig) {
    this.broadcast('config-changes', {
      type: 'config-changed',
      data: {
        section: configSection,
        config: newConfig,
        timestamp: new Date().toISOString()
      }
    });
  }

  broadcastLogEntry(logEntry) {
    this.broadcast('logs', {
      type: 'log-entry',
      data: logEntry,
      timestamp: new Date().toISOString()
    });
  }

  broadcastAlert(alert) {
    this.broadcast('alerts', {
      type: 'new-alert',
      data: alert,
      timestamp: new Date().toISOString()
    });
  }

  broadcastSessionEvent(event) {
    this.broadcast('session-limits', {
      type: 'session-event',
      data: event,
      timestamp: new Date().toISOString()
    });
  }

  // Client management
  getConnectedClients() {
    const clients = [];
    for (const [clientId, client] of this.clients.entries()) {
      clients.push({
        id: clientId,
        connected: client.ws.readyState === client.ws.OPEN,
        subscriptions: Array.from(client.subscriptions),
        lastActivity: client.lastActivity,
        ipAddress: client.ipAddress
      });
    }
    return clients;
  }

  getSubscriptionStats() {
    const stats = {};
    for (const [channel, subscribers] of this.subscriptions.entries()) {
      stats[channel] = subscribers.size;
    }
    return stats;
  }

  disconnectClient(clientId) {
    const client = this.clients.get(clientId);
    if (client && client.ws.readyState === client.ws.OPEN) {
      client.ws.close();
    }
  }

  // Cleanup dead connections
  cleanup() {
    const deadClients = [];
    
    for (const [clientId, client] of this.clients.entries()) {
      if (client.ws.readyState !== client.ws.OPEN) {
        deadClients.push(clientId);
      } else {
        // Check for inactive clients (no activity in 5 minutes)
        const inactive = new Date() - client.lastActivity > 5 * 60 * 1000;
        if (inactive) {
          deadClients.push(clientId);
        }
      }
    }

    for (const clientId of deadClients) {
      this.handleDisconnect(clientId);
    }

    if (deadClients.length > 0) {
      console.log(`Cleaned up ${deadClients.length} dead WebSocket connections`);
    }
  }
}

module.exports = { WebSocketManager };
