// Session Management JavaScript
class SessionManager {
    constructor(app) {
        this.app = app;
        this.currentSessions = [];
        this.currentLimits = {};
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Refresh sessions button
        const refreshBtn = document.getElementById('refreshSessions');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refreshSessions());
        }

        // Terminate session buttons (delegated)
        const sessionsTable = document.getElementById('sessionsTable');
        if (sessionsTable) {
            sessionsTable.addEventListener('click', (e) => {
                if (e.target.classList.contains('terminate-session-btn')) {
                    const sessionId = e.target.getAttribute('data-session-id');
                    this.terminateSession(sessionId);
                }
            });
        }

        // Session limits form
        const limitsForm = document.getElementById('sessionLimitsForm');
        if (limitsForm) {
            limitsForm.addEventListener('submit', (e) => this.handleLimitsFormSubmit(e));
        }

        // Per-user limits modal
        const addUserLimitBtn = document.getElementById('addUserLimit');
        if (addUserLimitBtn) {
            addUserLimitBtn.addEventListener('click', () => this.showAddUserLimitModal());
        }

        // User limit form
        const userLimitForm = document.getElementById('userLimitForm');
        if (userLimitForm) {
            userLimitForm.addEventListener('submit', (e) => this.handleUserLimitFormSubmit(e));
        }

        // Close user limit modal
        const closeUserLimitModal = document.getElementById('closeUserLimitModal');
        const cancelUserLimit = document.getElementById('cancelUserLimit');
        if (closeUserLimitModal) {
            closeUserLimitModal.addEventListener('click', () => this.closeUserLimitModal());
        }
        if (cancelUserLimit) {
            cancelUserLimit.addEventListener('click', () => this.closeUserLimitModal());
        }

        // Search sessions
        const searchInput = document.getElementById('sessionSearch');
        if (searchInput) {
            searchInput.addEventListener('input', () => this.filterSessions());
        }

        // Session status filter
        const statusFilter = document.getElementById('sessionStatusFilter');
        if (statusFilter) {
            statusFilter.addEventListener('change', () => this.filterSessions());
        }
    }

    async loadSessions() {
        try {
            await Promise.all([
                this.loadActiveSessions(),
                this.loadSessionLimits(),
                this.loadUserLimits(),
                this.loadSessionStatistics()
            ]);
        } catch (error) {
            console.error('Load sessions error:', error);
            this.app.showToast('error', 'Error', 'Failed to load session data');
        }
    }

    async loadActiveSessions() {
        try {
            const response = await this.app.apiCall('/api/sessions');
            if (response && response.success !== false) {
                this.currentSessions = response.sessions || [];
                this.populateSessionsTable(this.currentSessions);
            }
        } catch (error) {
            console.error('Load active sessions error:', error);
        }
    }

    async loadSessionLimits() {
        try {
            const response = await this.app.apiCall('/api/sessions/limits');
            if (response && response.success !== false) {
                this.currentLimits = response;
                this.populateSessionLimitsForm(response);
            }
        } catch (error) {
            console.error('Load session limits error:', error);
        }
    }

    async loadUserLimits() {
        try {
            const response = await this.app.apiCall('/api/sessions/user-limits');
            if (response && response.success !== false) {
                this.populateUserLimitsTable(response.userLimits || []);
            }
        } catch (error) {
            console.error('Load user limits error:', error);
        }
    }

    async loadSessionStatistics() {
        try {
            const response = await this.app.apiCall('/api/sessions/statistics');
            if (response && response.success !== false) {
                this.updateSessionStatistics(response);
            }
        } catch (error) {
            console.error('Load session statistics error:', error);
        }
    }

    populateSessionsTable(sessions) {
        const tbody = document.querySelector('#sessionsTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!sessions || sessions.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="text-center">No active sessions</td></tr>';
            return;
        }

        sessions.forEach(session => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${session.sessionId || 'N/A'}</td>
                <td>${session.username || 'N/A'}</td>
                <td>${session.userAgent || 'N/A'}</td>
                <td>${session.sourceIP || 'N/A'}</td>
                <td>${this.formatDuration(session.duration)}</td>
                <td><span class="status-badge status-${session.status}">${session.status}</span></td>
                <td>
                    <button class="btn btn-sm btn-danger terminate-session-btn" data-session-id="${session.sessionId}">
                        <i class="fas fa-times"></i> Terminate
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    populateSessionLimitsForm(limits) {
        this.app.setFormValue('enableSessionLimits', limits.enabled);
        this.app.setFormValue('maxSessionsPerUser', limits.maxSessionsPerUser);
        this.app.setFormValue('sessionLimitAction', limits.sessionLimitAction);
        this.app.setFormValue('sessionTimeout', limits.sessionTimeout);
        this.app.setFormValue('enableGlobalLimit', limits.enableGlobalLimit);
        this.app.setFormValue('maxGlobalSessions', limits.maxGlobalSessions);
    }

    populateUserLimitsTable(userLimits) {
        const tbody = document.querySelector('#userLimitsTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!userLimits || userLimits.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" class="text-center">No user-specific limits configured</td></tr>';
            return;
        }

        userLimits.forEach(limit => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${limit.username}</td>
                <td>${limit.maxSessions}</td>
                <td>${limit.currentSessions || 0}</td>
                <td>
                    <button class="btn btn-sm btn-secondary" onclick="sessions.editUserLimit('${limit.username}')">
                        <i class="fas fa-edit"></i> Edit
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="sessions.deleteUserLimit('${limit.username}')">
                        <i class="fas fa-trash"></i> Delete
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    updateSessionStatistics(stats) {
        this.app.updateElement('totalSessions', stats.totalSessions || 0);
        this.app.updateElement('activeSessions', stats.activeSessions || 0);
        this.app.updateElement('averageSessionDuration', this.formatDuration(stats.averageDuration));
        this.app.updateElement('sessionsToday', stats.sessionsToday || 0);
        this.app.updateElement('peakConcurrentSessions', stats.peakConcurrentSessions || 0);
        this.app.updateElement('sessionLimitViolations', stats.limitViolations || 0);
    }

    async handleLimitsFormSubmit(e) {
        e.preventDefault();

        const formData = new FormData(e.target);
        const limits = {
            enabled: formData.get('enableSessionLimits') === 'on',
            maxSessionsPerUser: parseInt(formData.get('maxSessionsPerUser')),
            sessionLimitAction: formData.get('sessionLimitAction'),
            sessionTimeout: parseInt(formData.get('sessionTimeout')),
            enableGlobalLimit: formData.get('enableGlobalLimit') === 'on',
            maxGlobalSessions: parseInt(formData.get('maxGlobalSessions'))
        };

        try {
            const response = await this.app.apiCall('/api/sessions/limits', {
                method: 'PUT',
                body: JSON.stringify(limits)
            });

            if (response && response.success) {
                this.app.showToast('success', 'Success', 'Session limits updated successfully');
                this.currentLimits = limits;
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to update session limits');
            }
        } catch (error) {
            console.error('Update session limits error:', error);
            this.app.showToast('error', 'Error', 'Failed to update session limits');
        }
    }

    async handleUserLimitFormSubmit(e) {
        e.preventDefault();

        const formData = new FormData(e.target);
        const username = formData.get('username');
        const maxSessions = parseInt(formData.get('maxSessions'));

        try {
            const response = await this.app.apiCall(`/api/sessions/limits/${username}`, {
                method: 'PUT',
                body: JSON.stringify({ limit: maxSessions })
            });

            if (response && response.success) {
                this.app.showToast('success', 'Success', 'User session limit updated successfully');
                this.closeUserLimitModal();
                this.loadUserLimits();
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to update user session limit');
            }
        } catch (error) {
            console.error('Update user session limit error:', error);
            this.app.showToast('error', 'Error', 'Failed to update user session limit');
        }
    }

    async terminateSession(sessionId) {
        if (!confirm('Are you sure you want to terminate this session?')) {
            return;
        }

        try {
            const response = await this.app.apiCall(`/api/sessions/${sessionId}/terminate`, {
                method: 'POST'
            });

            if (response && response.success) {
                this.app.showToast('success', 'Success', 'Session terminated successfully');
                this.loadActiveSessions();
                this.loadSessionStatistics();
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to terminate session');
            }
        } catch (error) {
            console.error('Terminate session error:', error);
            this.app.showToast('error', 'Error', 'Failed to terminate session');
        }
    }

    async deleteUserLimit(username) {
        if (!confirm(`Are you sure you want to delete the session limit for user "${username}"?`)) {
            return;
        }

        try {
            const response = await this.app.apiCall(`/api/sessions/limits/${username}`, {
                method: 'DELETE'
            });

            if (response && response.success) {
                this.app.showToast('success', 'Success', 'User session limit deleted successfully');
                this.loadUserLimits();
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to delete user session limit');
            }
        } catch (error) {
            console.error('Delete user session limit error:', error);
            this.app.showToast('error', 'Error', 'Failed to delete user session limit');
        }
    }

    async editUserLimit(username) {
        try {
            const response = await this.app.apiCall(`/api/sessions/limits/${username}`);
            if (response && response.success !== false) {
                this.app.setFormValue('userLimitUsername', username);
                this.app.setFormValue('userLimitMaxSessions', response.limit);
                this.showUserLimitModal();
            }
        } catch (error) {
            console.error('Load user session limit error:', error);
            this.app.showToast('error', 'Error', 'Failed to load user session limit');
        }
    }

    showAddUserLimitModal() {
        document.getElementById('userLimitForm').reset();
        this.showUserLimitModal();
    }

    showUserLimitModal() {
        const modal = document.getElementById('userLimitModal');
        if (modal) {
            modal.classList.add('active');
        }
    }

    closeUserLimitModal() {
        const modal = document.getElementById('userLimitModal');
        if (modal) {
            modal.classList.remove('active');
        }
    }

    filterSessions() {
        const searchTerm = document.getElementById('sessionSearch')?.value.toLowerCase() || '';
        const statusFilter = document.getElementById('sessionStatusFilter')?.value || 'all';

        let filteredSessions = this.currentSessions;

        // Apply search filter
        if (searchTerm) {
            filteredSessions = filteredSessions.filter(session => 
                (session.username && session.username.toLowerCase().includes(searchTerm)) ||
                (session.sessionId && session.sessionId.toLowerCase().includes(searchTerm)) ||
                (session.sourceIP && session.sourceIP.includes(searchTerm)) ||
                (session.userAgent && session.userAgent.toLowerCase().includes(searchTerm))
            );
        }

        // Apply status filter
        if (statusFilter !== 'all') {
            filteredSessions = filteredSessions.filter(session => 
                session.status === statusFilter
            );
        }

        this.populateSessionsTable(filteredSessions);
    }

    async refreshSessions() {
        const refreshBtn = document.getElementById('refreshSessions');
        if (refreshBtn) {
            refreshBtn.disabled = true;
            refreshBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Refreshing...';
        }

        try {
            await this.loadSessions();
            this.app.showToast('success', 'Refreshed', 'Session data updated');
        } catch (error) {
            console.error('Refresh sessions error:', error);
            this.app.showToast('error', 'Error', 'Failed to refresh session data');
        } finally {
            if (refreshBtn) {
                refreshBtn.disabled = false;
                refreshBtn.innerHTML = '<i class="fas fa-sync-alt"></i> Refresh';
            }
        }
    }

    formatDuration(seconds) {
        if (!seconds) return '0s';
        
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        if (hours > 0) {
            return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        }
        return `${minutes}:${secs.toString().padStart(2, '0')}`;
    }

    async terminateUserSessions(username) {
        if (!confirm(`Are you sure you want to terminate all sessions for user "${username}"?`)) {
            return;
        }

        try {
            const response = await this.app.apiCall(`/api/sessions/user/${username}/terminate`, {
                method: 'POST'
            });

            if (response && response.success) {
                this.app.showToast('success', 'Success', `All sessions for user "${username}" terminated successfully`);
                this.loadActiveSessions();
                this.loadSessionStatistics();
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to terminate user sessions');
            }
        } catch (error) {
            console.error('Terminate user sessions error:', error);
            this.app.showToast('error', 'Error', 'Failed to terminate user sessions');
        }
    }

    async exportSessionsReport() {
        try {
            const response = await this.app.apiCall('/api/sessions/export', {
                method: 'POST'
            });

            if (response && response.success) {
                // Create download link for report
                const blob = new Blob([JSON.stringify(response.report, null, 2)], {
                    type: 'application/json'
                });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `sessions-report-${new Date().toISOString().split('T')[0]}.json`;
                a.click();
                URL.revokeObjectURL(url);

                this.app.showToast('success', 'Success', 'Sessions report exported successfully');
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to export sessions report');
            }
        } catch (error) {
            console.error('Export sessions report error:', error);
            this.app.showToast('error', 'Error', 'Failed to export sessions report');
        }
    }
}

// Initialize session manager when DOM is loaded
let sessions;
document.addEventListener('DOMContentLoaded', () => {
    if (window.app) {
        sessions = new SessionManager(window.app);
    }
});
