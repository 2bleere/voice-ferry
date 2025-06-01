// Main Application Class
class VoiceFerryApp {
    constructor() {
        this.currentUser = null;
        this.authToken = null;
        this.currentPage = 'dashboard';
        this.isAuthenticated = false;
        this.websocket = null;
        
        this.init();
    }

    init() {
        // Check for existing authentication
        this.authToken = localStorage.getItem('auth_token');
        if (this.authToken) {
            this.validateToken();
        } else {
            this.showLogin();
        }

        // Set up event listeners
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Login form
        const loginForm = document.getElementById('loginForm');
        if (loginForm) {
            loginForm.addEventListener('submit', (e) => this.handleLogin(e));
        }

        // Logout button
        const logoutBtn = document.getElementById('logoutBtn');
        if (logoutBtn) {
            logoutBtn.addEventListener('click', () => this.handleLogout());
        }

        // Navigation
        const navLinks = document.querySelectorAll('.nav-link');
        navLinks.forEach(link => {
            link.addEventListener('click', (e) => this.handleNavigation(e));
        });

        // Page refresh buttons
        this.setupRefreshButtons();

        // Configuration save button
        const saveConfigBtn = document.getElementById('saveConfig');
        if (saveConfigBtn) {
            saveConfigBtn.addEventListener('click', () => this.saveConfiguration());
        }

        // Backup config button
        const backupConfigBtn = document.getElementById('backupConfig');
        if (backupConfigBtn) {
            backupConfigBtn.addEventListener('click', () => this.backupConfiguration());
        }

        // Configuration tabs
        const tabButtons = document.querySelectorAll('.tab-button');
        tabButtons.forEach(button => {
            button.addEventListener('click', (e) => this.switchConfigTab(e));
        });

        // Routing page buttons
        const addRoutingRuleBtn = document.getElementById('addRoutingRule');
        if (addRoutingRuleBtn) {
            addRoutingRuleBtn.addEventListener('click', () => this.editRoutingRule());
        }

        const testRoutingBtn = document.getElementById('testRouting');
        if (testRoutingBtn) {
            testRoutingBtn.addEventListener('click', () => {
                document.getElementById('routingTestModal').classList.add('active');
            });
        }

        // Routing modals
        const routingRuleModal = document.getElementById('routingRuleModal');
        const routingTestModal = document.getElementById('routingTestModal');

        // Routing rule modal events
        if (routingRuleModal) {
            const closeBtn = document.getElementById('closeRoutingRuleModal');
            const cancelBtn = document.getElementById('cancelRoutingRule');
            const saveBtn = document.getElementById('saveRoutingRule');
            const form = document.getElementById('routingRuleForm');

            if (closeBtn) closeBtn.addEventListener('click', () => routingRuleModal.classList.remove('active'));
            if (cancelBtn) cancelBtn.addEventListener('click', () => routingRuleModal.classList.remove('active'));
            if (form) {
                form.addEventListener('submit', (e) => {
                    e.preventDefault();
                    this.saveRoutingRule();
                });
            }
        }

        // Routing test modal events
        if (routingTestModal) {
            const closeBtn = document.getElementById('closeRoutingTestModal');
            const cancelBtn = document.getElementById('cancelRoutingTest');
            const testBtn = document.getElementById('runRoutingTest');
            const form = document.getElementById('routingTestForm');

            if (closeBtn) closeBtn.addEventListener('click', () => routingTestModal.classList.remove('active'));
            if (cancelBtn) cancelBtn.addEventListener('click', () => routingTestModal.classList.remove('active'));
            if (form) {
                form.addEventListener('submit', (e) => {
                    e.preventDefault();
                    this.testRouting();
                });
            }
        }
    }

    setupRefreshButtons() {
        const refreshButtons = [
            { id: 'refreshDashboard', handler: () => this.refreshDashboard() },
            { id: 'refreshSessions', handler: () => this.refreshSessions() },
            { id: 'refreshSipUsers', handler: () => this.refreshSipUsers() },
            { id: 'refreshCalls', handler: () => this.refreshCalls() },
            { id: 'refreshRouting', handler: () => this.loadRouting() }
        ];

        refreshButtons.forEach(({ id, handler }) => {
            const button = document.getElementById(id);
            if (button) {
                button.addEventListener('click', handler);
            }
        });
    }

    // Refresh methods that delegate to appropriate managers
    async refreshDashboard() {
        if (window.dashboard) {
            await window.dashboard.refreshDashboard();
        } else {
            await this.loadDashboard();
        }
    }

    async refreshSessions() {
        if (window.sessions) {
            await window.sessions.refreshSessions();
        }
    }

    async refreshSipUsers() {
        if (window.sipUsersManager) {
            await window.sipUsersManager.refreshUsers();
        }
    }

    async refreshCalls() {
        if (window.calls) {
            await window.calls.refreshCalls();
        }
    }

    async handleLogin(e) {
        e.preventDefault();
        
        const username = document.getElementById('username').value;
        const password = document.getElementById('password').value;
        const errorElement = document.getElementById('loginError');

        try {
            const response = await fetch('/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ username, password })
            });

            const data = await response.json();

            if (data.success) {
                this.authToken = data.token;
                this.currentUser = data.user;
                localStorage.setItem('auth_token', this.authToken);
                
                this.showApp();
                this.loadDashboard();
                
                this.showToast('success', 'Login Successful', 'Welcome back!');
            } else {
                errorElement.textContent = data.error || 'Login failed';
                errorElement.style.display = 'block';
            }
        } catch (error) {
            console.error('Login error:', error);
            errorElement.textContent = 'Connection error. Please try again.';
            errorElement.style.display = 'block';
        }
    }

    async validateToken() {
        try {
            const response = await fetch('/api/auth/me', {
                headers: {
                    'Authorization': `Bearer ${this.authToken}`
                }
            });

            if (response.ok) {
                const data = await response.json();
                this.currentUser = data.user;
                this.showApp();
                this.loadDashboard();
            } else {
                localStorage.removeItem('auth_token');
                this.showLogin();
            }
        } catch (error) {
            console.error('Token validation error:', error);
            localStorage.removeItem('auth_token');
            this.showLogin();
        }
    }

    async handleLogout() {
        try {
            await fetch('/api/auth/logout', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.authToken}`
                }
            });
        } catch (error) {
            console.error('Logout error:', error);
        }

        this.authToken = null;
        this.currentUser = null;
        localStorage.removeItem('auth_token');
        
        if (this.websocket) {
            this.websocket.close();
        }

        this.showLogin();
        this.showToast('info', 'Logged Out', 'You have been logged out successfully.');
    }

    showLogin() {
        document.getElementById('loginModal').classList.add('active');
        document.getElementById('app').style.display = 'none';
        this.isAuthenticated = false;
    }

    showApp() {
        document.getElementById('loginModal').classList.remove('active');
        document.getElementById('app').style.display = 'block';
        this.isAuthenticated = true;
        
        // Update user display
        const currentUserElement = document.getElementById('currentUser');
        if (currentUserElement && this.currentUser) {
            currentUserElement.textContent = this.currentUser.username;
        }
        
        // Initialize WebSocket connection
        this.initializeWebSocket();
    }

    initializeWebSocket() {
        if (window.websocketManager) {
            // WebSocket manager will handle connection automatically
            console.log('WebSocket manager initialized');
        }
    }

    handleNavigation(e) {
        e.preventDefault();
        
        const link = e.currentTarget;
        const page = link.getAttribute('data-page');
        
        // Update active nav link
        document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
        link.classList.add('active');
        
        // Show page
        this.showPage(page);
    }

    showPage(pageName) {
        // Hide all pages
        document.querySelectorAll('.page').forEach(page => page.classList.remove('active'));
        
        // Show selected page
        const targetPage = document.getElementById(pageName);
        if (targetPage) {
            targetPage.classList.add('active');
            this.currentPage = pageName;
            
            // Load page data
            this.loadPageData(pageName);
        }
    }

    async loadPageData(pageName) {
        switch (pageName) {
            case 'dashboard':
                if (window.dashboard) {
                    await window.dashboard.loadDashboard();
                }
                break;
            case 'configuration':
                if (window.configManager) {
                    await window.configManager.loadConfiguration();
                }
                break;
            case 'sessions':
                if (window.sessions) {
                    await window.sessions.loadSessions();
                }
                break;
            case 'sip-users':
                if (window.sipUsersManager) {
                    await window.sipUsersManager.loadUsers();
                }
                break;
            case 'calls':
                if (window.calls) {
                    await window.calls.loadCalls();
                }
                break;
            case 'metrics':
                await this.loadMetrics();
                break;
            case 'routing':
                await this.loadRouting();
                break;
            case 'logs':
                await this.loadLogs();
                break;
            case 'alerts':
                await this.loadAlerts();
                break;
        }
    }

    async loadDashboard() {
        // Dashboard loading is now handled by DashboardManager
        if (window.dashboard) {
            await window.dashboard.loadDashboard();
        } else {
            // Fallback for backward compatibility
            try {
                const [overview, status] = await Promise.all([
                    this.apiCall('/api/dashboard/overview'),
                    this.apiCall('/api/dashboard/status')
                ]);

                this.updateDashboardOverview(overview);
                this.updateSystemStatus(status);
            } catch (error) {
                console.error('Dashboard load error:', error);
                this.showToast('error', 'Error', 'Failed to load dashboard data');
            }
        }
    }

    updateDashboardOverview(data) {
        if (!data) return;

        // Update call statistics
        if (data.callStats) {
            this.updateElement('activeCalls', data.callStats.activeCalls || 0);
            this.updateElement('totalCalls', data.callStats.totalCalls || 0);
            this.updateElement('callSuccessRate', `${(data.callStats.successRate || 0).toFixed(1)}%`);
            this.updateElement('avgCallDuration', `${data.callStats.averageDuration || 0}s`);
        }

        // Update session statistics
        if (data.sessionStats) {
            this.updateElement('activeSessions', data.sessionStats.activeSessions || 0);
            this.updateElement('registeredUsers', data.sessionStats.registeredUsers || 0);
            this.updateElement('sessionLimitExceeded', data.sessionStats.limitExceeded || 0);
            this.updateElement('sessionUtilization', `${(data.sessionStats.utilization || 0).toFixed(1)}%`);
        }

        // Update performance metrics
        if (data.performanceStats) {
            this.updateElement('responseTime', `${(data.performanceStats.responseTime || 0).toFixed(0)}ms`);
            this.updateElement('throughput', `${data.performanceStats.throughput || 0}/s`);
            this.updateElement('errorRate', `${(data.performanceStats.errorRate || 0).toFixed(1)}%`);
            this.updateElement('availability', `${(data.performanceStats.availability || 100).toFixed(1)}%`);
        }

        // Update recent events
        if (data.events) {
            this.updateRecentEvents(data.events);
        }
    }

    updateSystemStatus(data) {
        if (!data) return;

        const statusMap = {
            'healthy': { class: 'status-healthy', text: 'Healthy' },
            'warning': { class: 'status-warning', text: 'Warning' },
            'error': { class: 'status-error', text: 'Error' },
            'degraded': { class: 'status-warning', text: 'Degraded' }
        };

        // Update overall system status
        const systemStatus = document.getElementById('systemStatus');
        if (systemStatus && data.overall) {
            const status = statusMap[data.overall] || statusMap.error;
            const dot = systemStatus.querySelector('.status-dot');
            const text = systemStatus.querySelector('.status-text');
            
            dot.className = `status-dot ${status.class}`;
            text.textContent = `System ${status.text}`;
        }

        // Update individual service statuses
        if (data.services) {
            this.updateServiceStatus('b2buaStatus', data.services.b2bua);
            this.updateServiceStatus('redisStatus', data.services.redis);
            this.updateServiceStatus('rtpengineStatus', data.services.rtpengine);
        }

        // Update uptime
        if (data.uptime) {
            this.updateElement('systemUptime', this.formatUptime(data.uptime));
        }
    }

    updateServiceStatus(elementId, status) {
        const element = document.getElementById(elementId);
        if (!element) return;

        const statusMap = {
            'healthy': { class: 'status-healthy', text: 'Healthy' },
            'unhealthy': { class: 'status-error', text: 'Unhealthy' },
            'connected': { class: 'status-healthy', text: 'Connected' },
            'disconnected': { class: 'status-error', text: 'Disconnected' },
            'active': { class: 'status-healthy', text: 'Active' },
            'inactive': { class: 'status-error', text: 'Inactive' }
        };

        const statusInfo = statusMap[status] || statusMap.unhealthy;
        const dot = element.querySelector('.status-dot');
        
        if (dot) {
            dot.className = `status-dot ${statusInfo.class}`;
        }
        
        // Update text content, preserving the dot
        const textParts = element.innerHTML.split('</span>');
        if (textParts.length > 1) {
            element.innerHTML = textParts[0] + '</span> ' + statusInfo.text;
        }
    }

    updateRecentEvents(events) {
        const container = document.getElementById('recentEvents');
        if (!container || !events) return;

        container.innerHTML = '';

        events.slice(0, 10).forEach(event => {
            const eventElement = document.createElement('div');
            eventElement.className = 'event-item';
            
            const severityClass = `event-severity-${event.severity}`;
            const iconMap = {
                info: 'fas fa-info-circle',
                warning: 'fas fa-exclamation-triangle',
                error: 'fas fa-exclamation-circle',
                critical: 'fas fa-times-circle'
            };

            eventElement.innerHTML = `
                <div class="event-icon ${severityClass}">
                    <i class="${iconMap[event.severity] || iconMap.info}"></i>
                </div>
                <div class="event-content">
                    <div class="event-message">${event.message}</div>
                    <div class="event-time">${this.formatTimestamp(event.timestamp)}</div>
                </div>
            `;

            container.appendChild(eventElement);
        });
    }

    async loadConfiguration() {
        try {
            const config = await this.apiCall('/api/config');
            this.populateConfigurationForm(config);
        } catch (error) {
            console.error('Configuration load error:', error);
            this.showToast('error', 'Error', 'Failed to load configuration');
        }
    }

    populateConfigurationForm(config) {
        if (!config) return;

        // SIP Configuration
        if (config.sip) {
            this.setFormValue('sipHost', config.sip.host);
            this.setFormValue('sipPort', config.sip.port);
            this.setFormValue('sipTransport', config.sip.transport);
            
            if (config.sip.timeouts) {
                this.setFormValue('transactionTimeout', config.sip.timeouts.transaction);
                this.setFormValue('dialogTimeout', config.sip.timeouts.dialog);
                this.setFormValue('registrationTimeout', config.sip.timeouts.registration);
            }
        }

        // Redis Configuration
        if (config.redis) {
            this.setFormValue('redisEnabled', config.redis.enabled);
            this.setFormValue('redisHost', config.redis.host);
            this.setFormValue('redisPort', config.redis.port);
            this.setFormValue('redisPassword', config.redis.password);
            this.setFormValue('redisDatabase', config.redis.database);
            this.setFormValue('sessionLimitsEnabled', config.redis.enable_session_limits);
            this.setFormValue('maxSessionsPerUser', config.redis.max_sessions_per_user);
            this.setFormValue('sessionLimitAction', config.redis.session_limit_action);
        }
    }

    async saveConfiguration() {
        try {
            const config = this.gatherConfigurationData();
            
            const response = await this.apiCall('/api/config/apply', {
                method: 'POST',
                body: JSON.stringify(config)
            });

            if (response.success) {
                this.showToast('success', 'Configuration Saved', 'Configuration has been applied successfully');
            } else {
                this.showToast('error', 'Save Failed', response.error || 'Failed to save configuration');
            }
        } catch (error) {
            console.error('Configuration save error:', error);
            this.showToast('error', 'Error', 'Failed to save configuration');
        }
    }

    gatherConfigurationData() {
        return {
            sip: {
                host: this.getFormValue('sipHost'),
                port: parseInt(this.getFormValue('sipPort')),
                transport: this.getFormValue('sipTransport'),
                timeouts: {
                    transaction: this.getFormValue('transactionTimeout'),
                    dialog: this.getFormValue('dialogTimeout'),
                    registration: this.getFormValue('registrationTimeout')
                }
            },
            redis: {
                enabled: this.getFormValue('redisEnabled', 'checkbox'),
                host: this.getFormValue('redisHost'),
                port: parseInt(this.getFormValue('redisPort')),
                password: this.getFormValue('redisPassword'),
                database: parseInt(this.getFormValue('redisDatabase')),
                enable_session_limits: this.getFormValue('sessionLimitsEnabled', 'checkbox'),
                max_sessions_per_user: parseInt(this.getFormValue('maxSessionsPerUser')),
                session_limit_action: this.getFormValue('sessionLimitAction')
            }
        };
    }

    async backupConfiguration() {
        try {
            const response = await this.apiCall('/api/config/backup', {
                method: 'POST',
                body: JSON.stringify({
                    description: `Manual backup - ${new Date().toISOString()}`
                })
            });

            if (response.success) {
                this.showToast('success', 'Backup Created', 'Configuration backup created successfully');
            } else {
                this.showToast('error', 'Backup Failed', 'Failed to create configuration backup');
            }
        } catch (error) {
            console.error('Backup error:', error);
            this.showToast('error', 'Error', 'Failed to create backup');
        }
    }

    switchConfigTab(e) {
        const button = e.currentTarget;
        const tabName = button.getAttribute('data-tab');
        
        // Update tab buttons
        document.querySelectorAll('.tab-button').forEach(btn => btn.classList.remove('active'));
        button.classList.add('active');
        
        // Update config sections
        document.querySelectorAll('.config-section').forEach(section => section.classList.remove('active'));
        const targetSection = document.getElementById(`${tabName}Config`);
        if (targetSection) {
            targetSection.classList.add('active');
        }
    }

    async loadSessions() {
        try {
            const sessions = await this.apiCall('/api/sessions/users');
            this.populateSessionsTable(sessions.users || []);
        } catch (error) {
            console.error('Sessions load error:', error);
            this.showToast('error', 'Error', 'Failed to load session data');
        }
    }

    populateSessionsTable(sessions) {
        const tbody = document.querySelector('#sessionsTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        sessions.forEach(session => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${session.username}</td>
                <td>${session.activeSessions}</td>
                <td>${session.limit}</td>
                <td>${this.formatTimestamp(session.lastLogin)}</td>
                <td>${session.ipAddress || 'N/A'}</td>
                <td>
                    <button class="btn btn-secondary btn-sm" onclick="app.viewUserSessions('${session.username}')">
                        <i class="fas fa-eye"></i> View
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    async loadCalls() {
        try {
            const calls = await this.apiCall('/api/dashboard/active-calls');
            this.populateCallsTable(calls || []);
        } catch (error) {
            console.error('Calls load error:', error);
            this.showToast('error', 'Error', 'Failed to load call data');
        }
    }

    populateCallsTable(calls) {
        const tbody = document.querySelector('#callsTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        calls.forEach(call => {
            const row = document.createElement('tr');
            const duration = this.calculateDuration(call.startTime);
            
            row.innerHTML = `
                <td>${call.id}</td>
                <td>${call.from}</td>
                <td>${call.to}</td>
                <td>
                    <span class="status-indicator ${call.status === 'established' ? 'status-healthy' : 'status-warning'}">
                        ${call.status}
                    </span>
                </td>
                <td>${duration}</td>
                <td>${this.formatTimestamp(call.startTime)}</td>
                <td>
                    <button class="btn btn-error btn-sm" onclick="app.terminateCall('${call.id}')">
                        <i class="fas fa-phone-slash"></i> End
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    async refreshDashboard() {
        await this.loadDashboard();
        this.showToast('info', 'Refreshed', 'Dashboard data updated');
    }

    async refreshSessions() {
        await this.loadSessions();
        this.showToast('info', 'Refreshed', 'Session data updated');
    }

    async refreshSipUsers() {
        if (window.sipUsersManager) {
            await window.sipUsersManager.loadUsers();
            this.showToast('info', 'Refreshed', 'SIP users data updated');
        }
    }

    async refreshCalls() {
        await this.loadCalls();
        this.showToast('info', 'Refreshed', 'Call data updated');
    }

    // WebSocket Management
    initializeWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}`;
        
        this.websocket = new WebSocket(wsUrl);
        
        this.websocket.onopen = () => {
            console.log('WebSocket connected');
            this.subscribeToUpdates();
        };
        
        this.websocket.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleWebSocketMessage(message);
            } catch (error) {
                console.error('WebSocket message error:', error);
            }
        };
        
        this.websocket.onclose = () => {
            console.log('WebSocket disconnected');
            // Attempt to reconnect after 5 seconds
            setTimeout(() => {
                if (this.isAuthenticated) {
                    this.initializeWebSocket();
                }
            }, 5000);
        };
        
        this.websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    subscribeToUpdates() {
        if (!this.websocket || this.websocket.readyState !== WebSocket.OPEN) return;

        const subscriptions = ['dashboard', 'system-metrics', 'call-stats', 'active-calls', 'session-limits', 'alerts'];
        
        subscriptions.forEach(channel => {
            this.websocket.send(JSON.stringify({
                type: 'subscribe',
                channel: channel
            }));
        });
    }

    handleWebSocketMessage(message) {
        switch (message.type) {
            case 'dashboard-update':
                if (this.currentPage === 'dashboard') {
                    this.updateDashboardOverview(message.data);
                }
                break;
            case 'system-metrics':
                this.updateSystemStatus(message.data);
                break;
            case 'call-stats':
                if (this.currentPage === 'dashboard') {
                    this.updateCallStats(message.data);
                }
                break;
            case 'active-calls':
                if (this.currentPage === 'calls') {
                    this.populateCallsTable(message.data);
                }
                break;
            case 'session-limits':
                if (this.currentPage === 'sessions') {
                    this.updateSessionLimits(message.data);
                }
                break;
            case 'alerts':
                this.handleNewAlerts(message.data);
                break;
        }
    }

    // Utility Methods
    async apiCall(endpoint, options = {}) {
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.authToken}`
            }
        };

        const response = await fetch(endpoint, { ...defaultOptions, ...options });
        
        if (!response.ok) {
            if (response.status === 401) {
                this.handleLogout();
                throw new Error('Authentication required');
            }
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        return await response.json();
    }

    updateElement(id, value) {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = value;
        }
    }

    setFormValue(id, value) {
        const element = document.getElementById(id);
        if (!element) return;

        if (element.type === 'checkbox') {
            element.checked = value;
        } else {
            element.value = value || '';
        }
    }

    getFormValue(id, type = 'text') {
        const element = document.getElementById(id);
        if (!element) return '';

        if (type === 'checkbox') {
            return element.checked;
        }
        return element.value;
    }

    formatTimestamp(timestamp) {
        if (!timestamp) return 'N/A';
        return new Date(timestamp).toLocaleString();
    }

    formatUptime(seconds) {
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        return `${days}d ${hours}h ${minutes}m`;
    }

    calculateDuration(startTime) {
        if (!startTime) return '0s';
        const seconds = Math.floor((new Date() - new Date(startTime)) / 1000);
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        if (hours > 0) {
            return `${hours}h ${minutes}m ${secs}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${secs}s`;
        } else {
            return `${secs}s`;
        }
    }

    showToast(type, title, message) {
        const container = document.getElementById('toastContainer');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        
        const iconMap = {
            success: 'fas fa-check-circle',
            warning: 'fas fa-exclamation-triangle',
            error: 'fas fa-times-circle',
            info: 'fas fa-info-circle'
        };

        toast.innerHTML = `
            <div class="toast-icon">
                <i class="${iconMap[type] || iconMap.info}"></i>
            </div>
            <div class="toast-content">
                <div class="toast-title">${title}</div>
                <div class="toast-message">${message}</div>
            </div>
            <button class="toast-close" onclick="this.parentElement.remove()">
                <i class="fas fa-times"></i>
            </button>
        `;

        container.appendChild(toast);

        // Show toast
        requestAnimationFrame(() => {
            toast.classList.add('show');
        });

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (toast.parentElement) {
                toast.classList.remove('show');
                setTimeout(() => toast.remove(), 300);
            }
        }, 5000);
    }

    // Utility methods for managers
    updateElement(elementId, value) {
        const element = document.getElementById(elementId);
        if (element) {
            element.textContent = value;
        }
    }

    setFormValue(elementId, value) {
        const element = document.getElementById(elementId);
        if (element) {
            if (element.type === 'checkbox') {
                element.checked = value;
            } else {
                element.value = value;
            }
        }
    }

    getFormValue(elementId) {
        const element = document.getElementById(elementId);
        if (element) {
            if (element.type === 'checkbox') {
                return element.checked;
            } else {
                return element.value;
            }
        }
        return null;
    }

    // Placeholder methods for actions
    async viewUserSessions(username) {
        console.log('View sessions for:', username);
        // Implementation would show user session details
    }

    async terminateCall(callId) {
        if (confirm('Are you sure you want to terminate this call?')) {
            try {
                await this.apiCall(`/api/calls/${callId}`, { method: 'DELETE' });
                this.showToast('success', 'Call Terminated', 'Call has been terminated successfully');
                this.refreshCalls();
            } catch (error) {
                this.showToast('error', 'Error', 'Failed to terminate call');
            }
        }
    }

    // Load placeholder methods for other pages
    async loadMetrics() {
        console.log('Loading metrics...');
    }

    async loadRouting() {
        console.log('Loading routing...');
        try {
            const response = await fetch('/api/routing/rules', {
                headers: {
                    'Authorization': `Bearer ${this.authToken}`
                }
            });
            
            if (!response.ok) {
                throw new Error('Failed to fetch routing rules');
            }
            
            const rules = await response.json(); // This is directly an array
            this.updateRoutingDisplay({ rules }); // Wrap in object for consistency
        } catch (error) {
            console.error('Load routing error:', error);
            this.showToast('Failed to load routing rules', 'error');
        }
    }

    updateRoutingDisplay(data) {
        // Update statistics
        const rules = data.rules || [];
        const activeRules = rules.filter(rule => rule.enabled);
        const priorities = [...new Set(rules.map(rule => rule.priority))];
        const providers = [...new Set(rules.map(rule => rule.provider))];

        document.getElementById('totalRoutingRules').textContent = rules.length;
        document.getElementById('activeRoutingRules').textContent = activeRules.length;
        document.getElementById('routingPriorities').textContent = priorities.length;
        document.getElementById('routingProviders').textContent = providers.length;

        // Update table
        const tbody = document.querySelector('#routingRulesTable tbody');
        tbody.innerHTML = '';

        rules.forEach(rule => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${rule.priority}</td>
                <td>${rule.name}</td>
                <td><code>${rule.pattern}</code></td>
                <td><code>${rule.destination}</code></td>
                <td>${rule.provider}</td>
                <td>
                    <span class="status-badge ${rule.enabled ? 'status-active' : 'status-inactive'}">
                        ${rule.enabled ? 'Active' : 'Inactive'}
                    </span>
                </td>
                <td>
                    <button class="btn btn-sm btn-secondary" onclick="app.editRoutingRule('${rule.id}')">
                        <i class="fas fa-edit"></i>
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="app.deleteRoutingRule('${rule.id}')">
                        <i class="fas fa-trash"></i>
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    async loadLogs() {
        console.log('Loading logs...');
    }

    async loadAlerts() {
        console.log('Loading alerts...');
    }

    // Routing Management Methods
    async editRoutingRule(ruleId) {
        try {
            // If ruleId is provided, fetch the rule data for editing
            if (ruleId) {
                const response = await fetch(`/api/routing/rules`, {
                    headers: {
                        'Authorization': `Bearer ${this.authToken}`
                    }
                });
                
                if (!response.ok) {
                    throw new Error('Failed to fetch routing rules');
                }
                
                const data = await response.json();
                const rule = data.rules.find(r => r.id === ruleId);
                
                if (rule) {
                    // Populate form with existing data
                    document.getElementById('ruleName').value = rule.name || '';
                    document.getElementById('rulePriority').value = rule.priority || 10;
                    document.getElementById('rulePattern').value = rule.pattern || '';
                    document.getElementById('ruleDestination').value = rule.destination || '';
                    document.getElementById('ruleProvider').value = rule.provider || '';
                    document.getElementById('ruleDescription').value = rule.description || '';
                    document.getElementById('ruleEnabled').checked = rule.enabled || false;
                    
                    document.getElementById('routingRuleModalTitle').textContent = 'Edit Routing Rule';
                    document.getElementById('routingRuleForm').dataset.ruleId = ruleId;
                }
            } else {
                // Clear form for new rule
                document.getElementById('routingRuleForm').reset();
                document.getElementById('rulePriority').value = 10;
                document.getElementById('ruleEnabled').checked = true;
                document.getElementById('routingRuleModalTitle').textContent = 'Add Routing Rule';
                delete document.getElementById('routingRuleForm').dataset.ruleId;
            }
            
            // Show modal
            document.getElementById('routingRuleModal').classList.add('active');
        } catch (error) {
            console.error('Edit routing rule error:', error);
            this.showToast('Failed to load routing rule data', 'error');
        }
    }

    async saveRoutingRule() {
        try {
            const form = document.getElementById('routingRuleForm');
            const formData = new FormData(form);
            const ruleId = form.dataset.ruleId;
            
            const rule = {
                name: formData.get('ruleName'),
                priority: parseInt(formData.get('rulePriority')),
                pattern: formData.get('rulePattern'),
                destination: formData.get('ruleDestination'),
                provider: formData.get('ruleProvider'),
                description: formData.get('ruleDescription'),
                enabled: formData.get('ruleEnabled') === 'on'
            };
            
            const url = ruleId ? `/api/routing/rules/${ruleId}` : '/api/routing/rules';
            const method = ruleId ? 'PUT' : 'POST';
            
            const response = await fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.authToken}`
                },
                body: JSON.stringify(rule)
            });
            
            if (!response.ok) {
                throw new Error('Failed to save routing rule');
            }
            
            document.getElementById('routingRuleModal').classList.remove('active');
            this.showToast(`Routing rule ${ruleId ? 'updated' : 'created'} successfully`, 'success');
            this.loadRouting(); // Refresh the routing table
        } catch (error) {
            console.error('Save routing rule error:', error);
            this.showToast('Failed to save routing rule', 'error');
        }
    }

    async deleteRoutingRule(ruleId) {
        if (!confirm('Are you sure you want to delete this routing rule?')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/routing/rules/${ruleId}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${this.authToken}`
                }
            });
            
            if (!response.ok) {
                throw new Error('Failed to delete routing rule');
            }
            
            this.showToast('Routing rule deleted successfully', 'success');
            this.loadRouting(); // Refresh the routing table
        } catch (error) {
            console.error('Delete routing rule error:', error);
            this.showToast('Failed to delete routing rule', 'error');
        }
    }

    async testRouting() {
        try {
            const form = document.getElementById('routingTestForm');
            const formData = new FormData(form);
            
            let headers = {};
            const headersText = formData.get('testHeaders');
            if (headersText.trim()) {
                try {
                    headers = JSON.parse(headersText);
                } catch (e) {
                    this.showToast('Invalid JSON in headers field', 'error');
                    return;
                }
            }
            
            const testData = {
                fromUri: formData.get('testFromUri'),
                toUri: formData.get('testToUri'),
                headers: headers
            };
            
            const response = await fetch('/api/routing/test', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.authToken}`
                },
                body: JSON.stringify(testData)
            });
            
            if (!response.ok) {
                throw new Error('Failed to test routing');
            }
            
            const result = await response.json();
            
            // Display result
            const resultDiv = document.getElementById('routingTestResult');
            const outputDiv = document.getElementById('routingTestOutput');
            
            outputDiv.innerHTML = `
                <div class="test-result-item">
                    <strong>Matched Rule:</strong> ${result.matchedRule ? result.matchedRule.name : 'No rule matched'}
                </div>
                <div class="test-result-item">
                    <strong>Destination:</strong> <code>${result.destination || 'N/A'}</code>
                </div>
                <div class="test-result-item">
                    <strong>Provider:</strong> ${result.provider || 'N/A'}
                </div>
                <div class="test-result-item">
                    <strong>Success:</strong> ${result.success ? 'Yes' : 'No'}
                </div>
                ${result.error ? `<div class="test-result-item error"><strong>Error:</strong> ${result.error}</div>` : ''}
            `;
            
            resultDiv.style.display = 'block';
        } catch (error) {
            console.error('Test routing error:', error);
            this.showToast('Failed to test routing', 'error');
        }
    }
}

// Initialize the application
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new VoiceFerryApp();
    
    // Make app available globally
    window.app = app;
    
    // Initialize SIP Users Manager
    window.sipUsersManager = new SipUsersManager(app);
});
