// Dashboard Management JavaScript
class DashboardManager {
    constructor(app) {
        this.app = app;
        this.refreshInterval = null;
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Refresh button
        const refreshBtn = document.getElementById('refreshDashboard');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refreshDashboard());
        }

        // Auto-refresh toggle
        const autoRefreshToggle = document.getElementById('autoRefresh');
        if (autoRefreshToggle) {
            autoRefreshToggle.addEventListener('change', (e) => this.toggleAutoRefresh(e.target.checked));
        }

        // Time range selector
        const timeRangeSelect = document.getElementById('timeRange');
        if (timeRangeSelect) {
            timeRangeSelect.addEventListener('change', () => this.loadCallStatistics());
        }
    }

    async loadDashboard() {
        try {
            await Promise.all([
                this.loadOverview(),
                this.loadSystemStatus(),
                this.loadCallStatistics(),
                this.loadActiveCalls()
            ]);
        } catch (error) {
            console.error('Load dashboard error:', error);
            this.app.showToast('error', 'Error', 'Failed to load dashboard data');
        }
    }

    async loadOverview() {
        try {
            const response = await this.app.apiCall('/api/dashboard/overview');
            if (response && response.success !== false) {
                this.updateOverviewDisplay(response);
            }
        } catch (error) {
            console.error('Load overview error:', error);
        }
    }

    async loadSystemStatus() {
        try {
            const response = await this.app.apiCall('/api/dashboard/status');
            if (response && response.success !== false) {
                this.updateSystemStatus(response);
            }
        } catch (error) {
            console.error('Load system status error:', error);
        }
    }

    async loadCallStatistics() {
        try {
            const timeRange = document.getElementById('timeRange')?.value || '1h';
            const response = await this.app.apiCall(`/api/dashboard/call-stats?timeRange=${timeRange}`);
            if (response && response.success !== false) {
                this.updateCallStatistics(response);
            }
        } catch (error) {
            console.error('Load call statistics error:', error);
        }
    }

    async loadActiveCalls() {
        try {
            const response = await this.app.apiCall('/api/dashboard/active-calls');
            if (response && response.success !== false) {
                this.updateActiveCalls(response);
            }
        } catch (error) {
            console.error('Load active calls error:', error);
        }
    }

    updateOverviewDisplay(data) {
        // Update overview metrics
        this.app.updateElement('totalCalls', data.totalCalls || 0);
        this.app.updateElement('activeCalls', data.activeCalls || 0);
        this.app.updateElement('registeredUsers', data.registeredUsers || 0);
        this.app.updateElement('systemUptime', this.formatUptime(data.uptime));
        this.app.updateElement('cpuUsage', data.cpuUsage || 0);
        this.app.updateElement('memoryUsage', data.memoryUsage || 0);
        this.app.updateElement('diskUsage', data.diskUsage || 0);

        // Update progress bars
        this.updateProgressBar('cpuProgressBar', data.cpuUsage || 0);
        this.updateProgressBar('memoryProgressBar', data.memoryUsage || 0);
        this.updateProgressBar('diskProgressBar', data.diskUsage || 0);
    }

    updateSystemStatus(data) {
        const statusElement = document.getElementById('systemStatus');
        const statusDot = statusElement?.querySelector('.status-dot');
        const statusText = statusElement?.querySelector('.status-text');

        if (statusDot && statusText) {
            statusDot.className = `status-dot status-${data.overall || 'unknown'}`;
            statusText.textContent = this.getStatusText(data.overall);
        }

        // Update component statuses - map from data.services structure
        if (data.services) {
            this.updateComponentStatus('b2buaStatus', {
                status: data.services.b2bua,
                message: data.services.b2bua === 'healthy' ? 'Healthy' : 'Unhealthy'
            });
            this.updateComponentStatus('redisStatus', {
                status: data.services.redis,
                message: data.services.redis === 'healthy' ? 'Connected' : 'Disconnected'
            });
            this.updateComponentStatus('etcdStatus', {
                status: data.services.etcd,
                message: data.services.etcd === 'healthy' ? 'Connected' : 'Disconnected'
            });
            this.updateComponentStatus('rtpengineStatus', {
                status: data.services.rtpengine || 'healthy',
                message: (data.services.rtpengine || 'healthy') === 'healthy' ? 'Active' : 'Inactive'
            });
        }
    }

    updateComponentStatus(elementId, status) {
        const element = document.getElementById(elementId);
        if (element) {
            const dot = element.querySelector('.status-dot');
            const text = element.querySelector('.status-text');
            
            if (dot && text) {
                dot.className = `status-dot status-${status?.status || 'unknown'}`;
                text.textContent = status?.message || 'Unknown';
            }
        }
    }

    updateCallStatistics(data) {
        // Update call metrics
        this.app.updateElement('totalCallsInRange', data.totalCalls || 0);
        this.app.updateElement('successfulCalls', data.successfulCalls || 0);
        this.app.updateElement('failedCalls', data.failedCalls || 0);
        this.app.updateElement('averageCallDuration', this.formatDuration(data.averageDuration));
        this.app.updateElement('callSuccessRate', `${data.successRate || 0}%`);

        // Update charts if available
        if (data.hourlyStats) {
            this.updateCallChart(data.hourlyStats);
        }
    }

    updateActiveCalls(data) {
        const tbody = document.querySelector('#activeCallsTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!data.calls || data.calls.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="text-center">No active calls</td></tr>';
            return;
        }

        data.calls.forEach(call => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${call.callId || 'N/A'}</td>
                <td>${call.from || 'N/A'}</td>
                <td>${call.to || 'N/A'}</td>
                <td>${this.formatDuration(call.duration)}</td>
                <td><span class="status-badge status-${call.status}">${call.status}</span></td>
                <td>
                    <button class="btn btn-sm btn-danger" onclick="dashboard.terminateCall('${call.callId}')">
                        <i class="fas fa-phone-slash"></i> End
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    updateProgressBar(elementId, percentage) {
        const progressBar = document.getElementById(elementId);
        if (progressBar) {
            progressBar.style.width = `${percentage}%`;
            progressBar.setAttribute('aria-valuenow', percentage);
            
            // Update color based on percentage
            progressBar.className = `progress-bar ${this.getProgressBarClass(percentage)}`;
        }
    }

    getProgressBarClass(percentage) {
        if (percentage >= 90) return 'bg-danger';
        if (percentage >= 75) return 'bg-warning';
        return 'bg-success';
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

    formatUptime(seconds) {
        if (!seconds) return '0s';
        
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        
        if (days > 0) return `${days}d ${hours}h ${minutes}m`;
        if (hours > 0) return `${hours}h ${minutes}m`;
        return `${minutes}m`;
    }

    formatDuration(seconds) {
        if (!seconds) return '0s';
        
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        if (hours > 0) return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
        return `${minutes}:${secs.toString().padStart(2, '0')}`;
    }

    async terminateCall(callId) {
        if (!confirm('Are you sure you want to terminate this call?')) {
            return;
        }

        try {
            const response = await this.app.apiCall(`/api/calls/${callId}/terminate`, {
                method: 'POST'
            });

            if (response.success) {
                this.app.showToast('success', 'Success', 'Call terminated successfully');
                this.loadActiveCalls();
            } else {
                this.app.showToast('error', 'Error', response.error || 'Failed to terminate call');
            }
        } catch (error) {
            console.error('Terminate call error:', error);
            this.app.showToast('error', 'Error', 'Failed to terminate call');
        }
    }

    toggleAutoRefresh(enabled) {
        if (enabled) {
            this.refreshInterval = setInterval(() => {
                this.refreshDashboard();
            }, 30000); // Refresh every 30 seconds
        } else {
            if (this.refreshInterval) {
                clearInterval(this.refreshInterval);
                this.refreshInterval = null;
            }
        }
    }

    async refreshDashboard() {
        const refreshBtn = document.getElementById('refreshDashboard');
        if (refreshBtn) {
            refreshBtn.disabled = true;
            refreshBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Refreshing...';
        }

        try {
            await this.loadDashboard();
            this.app.showToast('success', 'Refreshed', 'Dashboard data updated');
        } catch (error) {
            console.error('Refresh dashboard error:', error);
            this.app.showToast('error', 'Error', 'Failed to refresh dashboard');
        } finally {
            if (refreshBtn) {
                refreshBtn.disabled = false;
                refreshBtn.innerHTML = '<i class="fas fa-sync-alt"></i> Refresh';
            }
        }
    }

    updateCallChart(hourlyStats) {
        // Basic chart implementation - can be enhanced with Chart.js later
        const chartContainer = document.getElementById('callChart');
        if (!chartContainer) return;

        // Simple bar chart representation
        const maxCalls = Math.max(...hourlyStats.map(stat => stat.calls));
        const bars = hourlyStats.map(stat => {
            const height = maxCalls > 0 ? (stat.calls / maxCalls) * 100 : 0;
            return `
                <div class="chart-bar" style="height: ${height}%;" title="${stat.hour}: ${stat.calls} calls">
                    <div class="bar-value">${stat.calls}</div>
                </div>
            `;
        }).join('');

        chartContainer.innerHTML = `<div class="chart-bars">${bars}</div>`;
    }

    destroy() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }
}

// Initialize dashboard manager when DOM is loaded
let dashboard;
document.addEventListener('DOMContentLoaded', () => {
    if (window.app) {
        dashboard = new DashboardManager(window.app);
    }
});
