// Active Calls Management JavaScript
class CallsManager {
    constructor(app) {
        this.app = app;
        this.currentCalls = [];
        this.callHistory = [];
        this.refreshInterval = null;
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Refresh calls button
        const refreshBtn = document.getElementById('refreshCalls');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refreshCalls());
        }

        // Auto-refresh toggle
        const autoRefreshToggle = document.getElementById('autoRefreshCalls');
        if (autoRefreshToggle) {
            autoRefreshToggle.addEventListener('change', (e) => this.toggleAutoRefresh(e.target.checked));
        }

        // Terminate call buttons (delegated)
        const callsTable = document.getElementById('activeCallsTable');
        if (callsTable) {
            callsTable.addEventListener('click', (e) => {
                if (e.target.classList.contains('terminate-call-btn')) {
                    const callId = e.target.getAttribute('data-call-id');
                    this.terminateCall(callId);
                }
                if (e.target.classList.contains('view-call-btn')) {
                    const callId = e.target.getAttribute('data-call-id');
                    this.viewCallDetails(callId);
                }
            });
        }

        // Call history table
        const historyTable = document.getElementById('callHistoryTable');
        if (historyTable) {
            historyTable.addEventListener('click', (e) => {
                if (e.target.classList.contains('view-history-btn')) {
                    const callId = e.target.getAttribute('data-call-id');
                    this.viewCallDetails(callId);
                }
            });
        }

        // Search calls
        const searchInput = document.getElementById('callSearch');
        if (searchInput) {
            searchInput.addEventListener('input', () => this.filterCalls());
        }

        // Call status filter
        const statusFilter = document.getElementById('callStatusFilter');
        if (statusFilter) {
            statusFilter.addEventListener('change', () => this.filterCalls());
        }

        // Call direction filter
        const directionFilter = document.getElementById('callDirectionFilter');
        if (directionFilter) {
            directionFilter.addEventListener('change', () => this.filterCalls());
        }

        // Date range filter
        const dateFromInput = document.getElementById('callDateFrom');
        const dateToInput = document.getElementById('callDateTo');
        if (dateFromInput) {
            dateFromInput.addEventListener('change', () => this.filterCallHistory());
        }
        if (dateToInput) {
            dateToInput.addEventListener('change', () => this.filterCallHistory());
        }

        // Close call details modal
        const closeCallModal = document.getElementById('closeCallModal');
        if (closeCallModal) {
            closeCallModal.addEventListener('click', () => this.closeCallDetailsModal());
        }

        // Tab switching
        const tabButtons = document.querySelectorAll('.calls-tab');
        tabButtons.forEach(button => {
            button.addEventListener('click', (e) => this.switchTab(e));
        });
    }

    async loadCalls() {
        try {
            await Promise.all([
                this.loadActiveCalls(),
                this.loadCallStatistics(),
                this.loadCallHistory()
            ]);
        } catch (error) {
            console.error('Load calls error:', error);
            this.app.showToast('error', 'Error', 'Failed to load call data');
        }
    }

    async loadActiveCalls() {
        try {
            const response = await this.app.apiCall('/api/calls/active');
            if (response && response.success !== false) {
                this.currentCalls = response.calls || [];
                this.populateActiveCallsTable(this.currentCalls);
            }
        } catch (error) {
            console.error('Load active calls error:', error);
        }
    }

    async loadCallStatistics() {
        try {
            const response = await this.app.apiCall('/api/calls/statistics');
            if (response && response.success !== false) {
                this.updateCallStatistics(response);
            }
        } catch (error) {
            console.error('Load call statistics error:', error);
        }
    }

    async loadCallHistory() {
        try {
            const params = new URLSearchParams({
                limit: 100,
                offset: 0
            });

            const dateFrom = document.getElementById('callDateFrom')?.value;
            const dateTo = document.getElementById('callDateTo')?.value;
            
            if (dateFrom) params.append('dateFrom', dateFrom);
            if (dateTo) params.append('dateTo', dateTo);

            const response = await this.app.apiCall(`/api/calls/history?${params}`);
            if (response && response.success !== false) {
                this.callHistory = response.calls || [];
                this.populateCallHistoryTable(this.callHistory);
            }
        } catch (error) {
            console.error('Load call history error:', error);
        }
    }

    populateActiveCallsTable(calls) {
        const tbody = document.querySelector('#activeCallsTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!calls || calls.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="text-center">No active calls</td></tr>';
            return;
        }

        calls.forEach(call => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${call.callId || 'N/A'}</td>
                <td>${call.from || 'N/A'}</td>
                <td>${call.to || 'N/A'}</td>
                <td><span class="direction-badge direction-${call.direction}">${call.direction}</span></td>
                <td>${this.formatDuration(call.duration)}</td>
                <td><span class="status-badge status-${call.status}">${call.status}</span></td>
                <td>${call.codec || 'N/A'}</td>
                <td>
                    <button class="btn btn-sm btn-info view-call-btn" data-call-id="${call.callId}">
                        <i class="fas fa-eye"></i> View
                    </button>
                    <button class="btn btn-sm btn-danger terminate-call-btn" data-call-id="${call.callId}">
                        <i class="fas fa-phone-slash"></i> End
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    populateCallHistoryTable(calls) {
        const tbody = document.querySelector('#callHistoryTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!calls || calls.length === 0) {
            tbody.innerHTML = '<tr><td colspan="9" class="text-center">No call history found</td></tr>';
            return;
        }

        calls.forEach(call => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${call.callId || 'N/A'}</td>
                <td>${call.from || 'N/A'}</td>
                <td>${call.to || 'N/A'}</td>
                <td><span class="direction-badge direction-${call.direction}">${call.direction}</span></td>
                <td>${new Date(call.startTime).toLocaleString()}</td>
                <td>${call.endTime ? new Date(call.endTime).toLocaleString() : 'N/A'}</td>
                <td>${this.formatDuration(call.duration)}</td>
                <td><span class="status-badge status-${call.endReason}">${call.endReason}</span></td>
                <td>
                    <button class="btn btn-sm btn-info view-history-btn" data-call-id="${call.callId}">
                        <i class="fas fa-eye"></i> Details
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    updateCallStatistics(stats) {
        this.app.updateElement('totalCallsToday', stats.totalCallsToday || 0);
        this.app.updateElement('activeCallsCount', stats.activeCallsCount || 0);
        this.app.updateElement('callsPerHour', stats.callsPerHour || 0);
        this.app.updateElement('averageCallDuration', this.formatDuration(stats.averageCallDuration));
        this.app.updateElement('callSuccessRate', `${stats.callSuccessRate || 0}%`);
        this.app.updateElement('peakConcurrentCalls', stats.peakConcurrentCalls || 0);

        // Update call distribution chart
        if (stats.callDistribution) {
            this.updateCallDistributionChart(stats.callDistribution);
        }
    }

    async terminateCall(callId) {
        if (!confirm('Are you sure you want to terminate this call?')) {
            return;
        }

        try {
            const response = await this.app.apiCall(`/api/calls/${callId}/terminate`, {
                method: 'POST'
            });

            if (response && response.success) {
                this.app.showToast('success', 'Success', 'Call terminated successfully');
                this.loadActiveCalls();
                this.loadCallStatistics();
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to terminate call');
            }
        } catch (error) {
            console.error('Terminate call error:', error);
            this.app.showToast('error', 'Error', 'Failed to terminate call');
        }
    }

    async viewCallDetails(callId) {
        try {
            const response = await this.app.apiCall(`/api/calls/${callId}/details`);
            if (response && response.success !== false) {
                this.populateCallDetailsModal(response.call);
                this.showCallDetailsModal();
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to load call details');
            }
        } catch (error) {
            console.error('Load call details error:', error);
            this.app.showToast('error', 'Error', 'Failed to load call details');
        }
    }

    populateCallDetailsModal(call) {
        // Populate basic call information
        this.app.updateElement('modalCallId', call.callId);
        this.app.updateElement('modalCallFrom', call.from);
        this.app.updateElement('modalCallTo', call.to);
        this.app.updateElement('modalCallDirection', call.direction);
        this.app.updateElement('modalCallStatus', call.status);
        this.app.updateElement('modalCallStartTime', new Date(call.startTime).toLocaleString());
        this.app.updateElement('modalCallEndTime', call.endTime ? new Date(call.endTime).toLocaleString() : 'N/A');
        this.app.updateElement('modalCallDuration', this.formatDuration(call.duration));

        // Populate technical details
        if (call.technical) {
            this.app.updateElement('modalCallCodec', call.technical.codec);
            this.app.updateElement('modalCallLocalSDP', call.technical.localSDP);
            this.app.updateElement('modalCallRemoteSDP', call.technical.remoteSDP);
            this.app.updateElement('modalCallUserAgent', call.technical.userAgent);
        }

        // Populate media statistics
        if (call.mediaStats) {
            this.app.updateElement('modalRtpPacketsSent', call.mediaStats.rtpPacketsSent);
            this.app.updateElement('modalRtpPacketsReceived', call.mediaStats.rtpPacketsReceived);
            this.app.updateElement('modalRtpBytesSent', this.formatBytes(call.mediaStats.rtpBytesSent));
            this.app.updateElement('modalRtpBytesReceived', this.formatBytes(call.mediaStats.rtpBytesReceived));
            this.app.updateElement('modalJitter', `${call.mediaStats.jitter || 0}ms`);
            this.app.updateElement('modalPacketLoss', `${call.mediaStats.packetLoss || 0}%`);
        }

        // Populate call events
        if (call.events) {
            this.populateCallEvents(call.events);
        }
    }

    populateCallEvents(events) {
        const eventsList = document.getElementById('modalCallEvents');
        if (!eventsList) return;

        eventsList.innerHTML = '';

        events.forEach(event => {
            const eventItem = document.createElement('div');
            eventItem.className = 'call-event-item';
            eventItem.innerHTML = `
                <div class="event-time">${new Date(event.timestamp).toLocaleString()}</div>
                <div class="event-type">${event.type}</div>
                <div class="event-description">${event.description}</div>
            `;
            eventsList.appendChild(eventItem);
        });
    }

    showCallDetailsModal() {
        const modal = document.getElementById('callDetailsModal');
        if (modal) {
            modal.classList.add('active');
        }
    }

    closeCallDetailsModal() {
        const modal = document.getElementById('callDetailsModal');
        if (modal) {
            modal.classList.remove('active');
        }
    }

    filterCalls() {
        const searchTerm = document.getElementById('callSearch')?.value.toLowerCase() || '';
        const statusFilter = document.getElementById('callStatusFilter')?.value || 'all';
        const directionFilter = document.getElementById('callDirectionFilter')?.value || 'all';

        let filteredCalls = this.currentCalls;

        // Apply search filter
        if (searchTerm) {
            filteredCalls = filteredCalls.filter(call => 
                (call.callId && call.callId.toLowerCase().includes(searchTerm)) ||
                (call.from && call.from.toLowerCase().includes(searchTerm)) ||
                (call.to && call.to.toLowerCase().includes(searchTerm))
            );
        }

        // Apply status filter
        if (statusFilter !== 'all') {
            filteredCalls = filteredCalls.filter(call => call.status === statusFilter);
        }

        // Apply direction filter
        if (directionFilter !== 'all') {
            filteredCalls = filteredCalls.filter(call => call.direction === directionFilter);
        }

        this.populateActiveCallsTable(filteredCalls);
    }

    filterCallHistory() {
        this.loadCallHistory();
    }

    switchTab(e) {
        e.preventDefault();
        
        const tabId = e.target.getAttribute('data-tab');
        
        // Update active tab
        document.querySelectorAll('.calls-tab').forEach(tab => {
            tab.classList.remove('active');
        });
        e.target.classList.add('active');
        
        // Show corresponding tab content
        document.querySelectorAll('.calls-tab-content').forEach(content => {
            content.classList.remove('active');
        });
        
        const targetContent = document.getElementById(`${tabId}Tab`);
        if (targetContent) {
            targetContent.classList.add('active');
        }

        // Load data for the active tab
        if (tabId === 'active') {
            this.loadActiveCalls();
        } else if (tabId === 'history') {
            this.loadCallHistory();
        }
    }

    toggleAutoRefresh(enabled) {
        if (enabled) {
            this.refreshInterval = setInterval(() => {
                this.loadActiveCalls();
                this.loadCallStatistics();
            }, 15000); // Refresh every 15 seconds
        } else {
            if (this.refreshInterval) {
                clearInterval(this.refreshInterval);
                this.refreshInterval = null;
            }
        }
    }

    async refreshCalls() {
        const refreshBtn = document.getElementById('refreshCalls');
        if (refreshBtn) {
            refreshBtn.disabled = true;
            refreshBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Refreshing...';
        }

        try {
            await this.loadCalls();
            this.app.showToast('success', 'Refreshed', 'Call data updated');
        } catch (error) {
            console.error('Refresh calls error:', error);
            this.app.showToast('error', 'Error', 'Failed to refresh call data');
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

    formatBytes(bytes) {
        if (!bytes) return '0 B';
        
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`;
    }

    updateCallDistributionChart(distribution) {
        const chartContainer = document.getElementById('callDistributionChart');
        if (!chartContainer) return;

        // Simple pie chart representation
        const total = distribution.reduce((sum, item) => sum + item.count, 0);
        
        const segments = distribution.map(item => {
            const percentage = total > 0 ? (item.count / total) * 100 : 0;
            return `
                <div class="chart-segment" style="--percentage: ${percentage}%;" title="${item.type}: ${item.count} calls (${percentage.toFixed(1)}%)">
                    <span class="segment-label">${item.type}</span>
                    <span class="segment-value">${item.count}</span>
                </div>
            `;
        }).join('');

        chartContainer.innerHTML = `<div class="chart-segments">${segments}</div>`;
    }

    async exportCallsReport() {
        try {
            const dateFrom = document.getElementById('callDateFrom')?.value;
            const dateTo = document.getElementById('callDateTo')?.value;
            
            const params = new URLSearchParams();
            if (dateFrom) params.append('dateFrom', dateFrom);
            if (dateTo) params.append('dateTo', dateTo);

            const response = await this.app.apiCall(`/api/calls/export?${params}`, {
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
                a.download = `calls-report-${new Date().toISOString().split('T')[0]}.json`;
                a.click();
                URL.revokeObjectURL(url);

                this.app.showToast('success', 'Success', 'Calls report exported successfully');
            } else {
                this.app.showToast('error', 'Error', response?.error || 'Failed to export calls report');
            }
        } catch (error) {
            console.error('Export calls report error:', error);
            this.app.showToast('error', 'Error', 'Failed to export calls report');
        }
    }

    destroy() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }
}

// Initialize calls manager when DOM is loaded
let calls;
document.addEventListener('DOMContentLoaded', () => {
    if (window.app) {
        calls = new CallsManager(window.app);
    }
});
