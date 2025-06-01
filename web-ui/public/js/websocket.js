// WebSocket Management JavaScript
class WebSocketManager {
    constructor(app) {
        this.app = app;
        this.websocket = null;
        this.reconnectInterval = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 1000; // Start with 1 second
        this.maxReconnectDelay = 30000; // Max 30 seconds
        this.isConnected = false;
        this.subscribers = new Map();
        
        this.init();
    }

    init() {
        this.connect();
        this.setupEventHandlers();
    }

    setupEventHandlers() {
        // Handle browser events
        window.addEventListener('beforeunload', () => {
            this.disconnect();
        });

        window.addEventListener('online', () => {
            if (!this.isConnected) {
                this.connect();
            }
        });

        window.addEventListener('offline', () => {
            this.updateConnectionStatus('offline');
        });
    }

    connect() {
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            return;
        }

        try {
            // Determine WebSocket URL
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            const wsUrl = `${protocol}//${host}/ws`;

            this.websocket = new WebSocket(wsUrl);
            
            this.websocket.onopen = (event) => {
                this.onOpen(event);
            };

            this.websocket.onmessage = (event) => {
                this.onMessage(event);
            };

            this.websocket.onclose = (event) => {
                this.onClose(event);
            };

            this.websocket.onerror = (event) => {
                this.onError(event);
            };

        } catch (error) {
            console.error('WebSocket connection error:', error);
            this.scheduleReconnect();
        }
    }

    onOpen(event) {
        console.log('WebSocket connected');
        this.isConnected = true;
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        
        // Clear any existing reconnect timer
        if (this.reconnectInterval) {
            clearTimeout(this.reconnectInterval);
            this.reconnectInterval = null;
        }

        this.updateConnectionStatus('connected');

        // Send authentication if we have a token
        if (this.app.authToken) {
            this.send({
                type: 'auth',
                token: this.app.authToken
            });
        }

        // Subscribe to default channels
        this.subscribeToDefaultChannels();

        this.app.showToast('success', 'Connected', 'Real-time updates enabled');
    }

    onMessage(event) {
        try {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
        } catch (error) {
            console.error('WebSocket message parse error:', error);
        }
    }

    onClose(event) {
        console.log('WebSocket disconnected:', event.code, event.reason);
        this.isConnected = false;
        this.updateConnectionStatus('disconnected');

        // Only attempt to reconnect if it wasn't a clean closure
        if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
        }
    }

    onError(error) {
        console.error('WebSocket error:', error);
        this.updateConnectionStatus('error');
    }

    handleMessage(message) {
        switch (message.type) {
            case 'auth_response':
                this.handleAuthResponse(message);
                break;
            case 'system_status':
                this.handleSystemStatus(message);
                break;
            case 'call_event':
                this.handleCallEvent(message);
                break;
            case 'session_event':
                this.handleSessionEvent(message);
                break;
            case 'sip_user_event':
                this.handleSipUserEvent(message);
                break;
            case 'alert':
                this.handleAlert(message);
                break;
            case 'metrics_update':
                this.handleMetricsUpdate(message);
                break;
            default:
                console.log('Unknown WebSocket message type:', message.type);
        }

        // Notify subscribers
        this.notifySubscribers(message.type, message);
    }

    handleAuthResponse(message) {
        if (message.success) {
            console.log('WebSocket authentication successful');
        } else {
            console.error('WebSocket authentication failed:', message.error);
            this.app.showToast('error', 'Auth Error', 'WebSocket authentication failed');
        }
    }

    handleSystemStatus(message) {
        // Update system status display
        if (window.dashboard) {
            window.dashboard.updateSystemStatus(message.data);
        }

        // Update header status indicator
        const statusElement = document.getElementById('systemStatus');
        if (statusElement) {
            const statusDot = statusElement.querySelector('.status-dot');
            const statusText = statusElement.querySelector('.status-text');
            
            if (statusDot && statusText) {
                statusDot.className = `status-dot status-${message.data.overall}`;
                statusText.textContent = this.getStatusText(message.data.overall);
            }
        }
    }

    handleCallEvent(message) {
        const { event, data } = message;
        
        switch (event) {
            case 'call_started':
                this.app.showToast('info', 'New Call', `Call started: ${data.from} → ${data.to}`);
                break;
            case 'call_ended':
                this.app.showToast('info', 'Call Ended', `Call ended: ${data.from} → ${data.to}`);
                break;
            case 'call_failed':
                this.app.showToast('warning', 'Call Failed', `Call failed: ${data.from} → ${data.to}`);
                break;
        }

        // Refresh calls data if on calls page
        if (this.app.currentPage === 'calls' && window.calls) {
            window.calls.loadActiveCalls();
            window.calls.loadCallStatistics();
        }

        // Update dashboard if active
        if (this.app.currentPage === 'dashboard' && window.dashboard) {
            window.dashboard.loadActiveCalls();
            window.dashboard.loadCallStatistics();
        }
    }

    handleSessionEvent(message) {
        const { event, data } = message;
        
        switch (event) {
            case 'session_registered':
                this.app.showToast('success', 'User Registered', `${data.username} registered`);
                break;
            case 'session_unregistered':
                this.app.showToast('info', 'User Unregistered', `${data.username} unregistered`);
                break;
            case 'session_limit_exceeded':
                this.app.showToast('warning', 'Session Limit', `Session limit exceeded for ${data.username}`);
                break;
        }

        // Refresh sessions data if on sessions page
        if (this.app.currentPage === 'sessions' && window.sessions) {
            window.sessions.loadActiveSessions();
            window.sessions.loadSessionStatistics();
        }
    }

    handleSipUserEvent(message) {
        const { event, data } = message;
        
        switch (event) {
            case 'user_created':
                this.app.showToast('success', 'User Created', `SIP user ${data.username} created`);
                break;
            case 'user_updated':
                this.app.showToast('info', 'User Updated', `SIP user ${data.username} updated`);
                break;
            case 'user_deleted':
                this.app.showToast('warning', 'User Deleted', `SIP user ${data.username} deleted`);
                break;
        }

        // Refresh SIP users data if on SIP users page
        if (this.app.currentPage === 'sip-users' && window.sipUsersManager) {
            window.sipUsersManager.loadUsers();
        }
    }

    handleAlert(message) {
        const { level, title, text } = message;
        this.app.showToast(level, title, text);

        // Store alert for alerts page
        this.storeAlert(message);
    }

    handleMetricsUpdate(message) {
        // Update metrics displays across the application
        if (window.dashboard && this.app.currentPage === 'dashboard') {
            window.dashboard.updateOverviewDisplay(message.data);
        }
    }

    send(message) {
        if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
            this.websocket.send(JSON.stringify(message));
        } else {
            console.warn('WebSocket not connected, message not sent:', message);
        }
    }

    subscribe(channel, callback) {
        if (!this.subscribers.has(channel)) {
            this.subscribers.set(channel, new Set());
        }
        this.subscribers.get(channel).add(callback);

        // Send subscription message
        this.send({
            type: 'subscribe',
            channel: channel
        });
    }

    unsubscribe(channel, callback) {
        if (this.subscribers.has(channel)) {
            this.subscribers.get(channel).delete(callback);
            
            if (this.subscribers.get(channel).size === 0) {
                this.subscribers.delete(channel);
                
                // Send unsubscription message
                this.send({
                    type: 'unsubscribe',
                    channel: channel
                });
            }
        }
    }

    notifySubscribers(channel, message) {
        if (this.subscribers.has(channel)) {
            this.subscribers.get(channel).forEach(callback => {
                try {
                    callback(message);
                } catch (error) {
                    console.error('Subscriber callback error:', error);
                }
            });
        }
    }

    subscribeToDefaultChannels() {
        const defaultChannels = [
            'system_status',
            'call_events',
            'session_events',
            'sip_user_events',
            'alerts',
            'metrics'
        ];

        defaultChannels.forEach(channel => {
            this.send({
                type: 'subscribe',
                channel: channel
            });
        });
    }

    scheduleReconnect() {
        if (this.reconnectInterval) {
            return; // Already scheduled
        }

        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1), this.maxReconnectDelay);

        console.log(`Scheduling WebSocket reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`);

        this.reconnectInterval = setTimeout(() => {
            this.reconnectInterval = null;
            this.connect();
        }, delay);
    }

    disconnect() {
        if (this.reconnectInterval) {
            clearTimeout(this.reconnectInterval);
            this.reconnectInterval = null;
        }

        if (this.websocket) {
            this.websocket.close(1000, 'Client disconnect');
            this.websocket = null;
        }

        this.isConnected = false;
        this.updateConnectionStatus('disconnected');
    }

    updateConnectionStatus(status) {
        const statusElement = document.getElementById('websocketStatus');
        if (statusElement) {
            const statusDot = statusElement.querySelector('.status-dot');
            const statusText = statusElement.querySelector('.status-text');
            
            if (statusDot && statusText) {
                statusDot.className = `status-dot status-${status}`;
                statusText.textContent = this.getConnectionStatusText(status);
            }
        }

        // Update connection indicator in header if present
        const connectionIndicator = document.getElementById('connectionIndicator');
        if (connectionIndicator) {
            connectionIndicator.className = `connection-indicator connection-${status}`;
            connectionIndicator.title = this.getConnectionStatusText(status);
        }
    }

    getStatusText(status) {
        const statusTexts = {
            healthy: 'System Healthy',
            warning: 'System Warning',
            error: 'System Error',
            unknown: 'Status Unknown'
        };
        return statusTexts[status] || 'Status Unknown';
    }

    getConnectionStatusText(status) {
        const statusTexts = {
            connected: 'Real-time updates active',
            disconnected: 'Real-time updates disconnected',
            error: 'Real-time connection error',
            offline: 'No internet connection'
        };
        return statusTexts[status] || 'Connection status unknown';
    }

    storeAlert(alert) {
        // Store alert in localStorage for alerts page
        let alerts = JSON.parse(localStorage.getItem('voice_ferry_alerts') || '[]');
        alerts.unshift({
            ...alert,
            id: Date.now(),
            timestamp: new Date().toISOString(),
            read: false
        });

        // Keep only last 100 alerts
        alerts = alerts.slice(0, 100);
        localStorage.setItem('voice_ferry_alerts', JSON.stringify(alerts));

        // Update alerts badge
        this.updateAlertsBadge();
    }

    updateAlertsBadge() {
        const alerts = JSON.parse(localStorage.getItem('voice_ferry_alerts') || '[]');
        const unreadCount = alerts.filter(alert => !alert.read).length;
        
        const badge = document.getElementById('alertsBadge');
        if (badge) {
            if (unreadCount > 0) {
                badge.textContent = unreadCount > 99 ? '99+' : unreadCount;
                badge.style.display = 'inline';
            } else {
                badge.style.display = 'none';
            }
        }
    }

    getConnectionState() {
        return {
            isConnected: this.isConnected,
            reconnectAttempts: this.reconnectAttempts,
            websocketState: this.websocket ? this.websocket.readyState : null
        };
    }

    // Manual reconnect method
    reconnect() {
        this.disconnect();
        this.reconnectAttempts = 0;
        this.connect();
    }
}

// Initialize WebSocket manager when DOM is loaded
let websocketManager;
document.addEventListener('DOMContentLoaded', () => {
    if (window.app) {
        websocketManager = new WebSocketManager(window.app);
        
        // Make it available globally
        window.websocketManager = websocketManager;
    }
});
