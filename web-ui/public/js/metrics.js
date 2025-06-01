// Metrics Page JavaScript
class MetricsManager {
    constructor() {
        this.currentTimeRange = '1h';
        this.refreshInterval = null;
        this.charts = {};
        this.init();
    }

    init() {
        this.attachEventListeners();
        this.setupTabs();
        this.loadMetrics();
        this.startAutoRefresh();
    }

    attachEventListeners() {
        // Time range selector
        const timeRangeSelect = document.getElementById('metricsTimeRange');
        if (timeRangeSelect) {
            timeRangeSelect.addEventListener('change', (e) => {
                this.currentTimeRange = e.target.value;
                this.loadMetrics();
            });
        }

        // Refresh button
        const refreshBtn = document.getElementById('refreshMetrics');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.loadMetrics();
            });
        }
    }

    setupTabs() {
        const tabButtons = document.querySelectorAll('.tab-button[data-tab]');
        tabButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                const tabId = e.target.getAttribute('data-tab');
                this.switchTab(tabId);
            });
        });
    }

    switchTab(tabId) {
        // Remove active class from all tabs and contents
        document.querySelectorAll('.tab-button').forEach(btn => btn.classList.remove('active'));
        document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));

        // Add active class to selected tab and content
        document.querySelector(`[data-tab="${tabId}"]`).classList.add('active');
        document.getElementById(tabId).classList.add('active');

        // Load data for the selected tab
        this.loadTabData(tabId);
    }

    async loadMetrics() {
        try {
            await Promise.all([
                this.loadSystemMetrics(),
                this.loadSipMetrics(),
                this.loadPerformanceMetrics(),
                this.loadRedisMetrics()
            ]);
        } catch (error) {
            console.error('Failed to load metrics:', error);
            showToast('Failed to load metrics', 'error');
        }
    }

    async loadSystemMetrics() {
        try {
            const response = await fetch(`/api/metrics/system?timeRange=${this.currentTimeRange}`);
            if (!response.ok) throw new Error('Failed to fetch system metrics');
            
            const data = await response.json();
            this.updateSystemMetrics(data);
        } catch (error) {
            console.error('System metrics error:', error);
        }
    }

    async loadSipMetrics() {
        try {
            const response = await fetch(`/api/metrics/sip?timeRange=${this.currentTimeRange}`);
            if (!response.ok) throw new Error('Failed to fetch SIP metrics');
            
            const data = await response.json();
            this.updateSipMetrics(data);
        } catch (error) {
            console.error('SIP metrics error:', error);
        }
    }

    async loadPerformanceMetrics() {
        try {
            const response = await fetch(`/api/metrics/performance?timeRange=${this.currentTimeRange}`);
            if (!response.ok) throw new Error('Failed to fetch performance metrics');
            
            const data = await response.json();
            this.updatePerformanceMetrics(data);
        } catch (error) {
            console.error('Performance metrics error:', error);
        }
    }

    async loadRedisMetrics() {
        try {
            const response = await fetch('/api/metrics/redis');
            if (!response.ok) throw new Error('Failed to fetch Redis metrics');
            
            const data = await response.json();
            this.updateRedisMetrics(data);
        } catch (error) {
            console.error('Redis metrics error:', error);
        }
    }

    updateSystemMetrics(data) {
        // Update CPU usage
        const cpuElement = document.getElementById('cpuUsage');
        if (cpuElement && data.cpu) {
            cpuElement.textContent = `${data.cpu.usage?.toFixed(1) || 0}%`;
            this.updateProgressBar('cpuChart', data.cpu.usage || 0);
        }

        // Update Memory usage
        const memoryElement = document.getElementById('memoryUsage');
        if (memoryElement && data.memory) {
            const memoryPercent = data.memory.used_percent || 0;
            memoryElement.textContent = `${memoryPercent.toFixed(1)}%`;
            this.updateProgressBar('memoryChart', memoryPercent);
        }

        // Update Network I/O
        if (data.network) {
            const rxElement = document.getElementById('networkRx');
            const txElement = document.getElementById('networkTx');
            if (rxElement) rxElement.textContent = this.formatBytes(data.network.bytes_recv || 0) + '/s';
            if (txElement) txElement.textContent = this.formatBytes(data.network.bytes_sent || 0) + '/s';
        }

        // Update Disk usage
        const diskElement = document.getElementById('diskUsage');
        const diskInfoElement = document.getElementById('diskInfo');
        if (diskElement && data.disk) {
            const diskPercent = data.disk.used_percent || 0;
            diskElement.textContent = `${diskPercent.toFixed(1)}%`;
            if (diskInfoElement) {
                const used = this.formatBytes(data.disk.used || 0);
                const total = this.formatBytes(data.disk.total || 0);
                diskInfoElement.textContent = `${used} used of ${total}`;
            }
        }
    }

    updateSipMetrics(data) {
        if (data.calls) {
            const totalCallsElement = document.getElementById('totalCalls');
            const activeCallsElement = document.getElementById('activeCalls');
            const successfulCallsElement = document.getElementById('successfulCalls');
            const failedCallsElement = document.getElementById('failedCalls');

            if (totalCallsElement) totalCallsElement.textContent = data.calls.total || 0;
            if (activeCallsElement) activeCallsElement.textContent = data.calls.active || 0;
            if (successfulCallsElement) successfulCallsElement.textContent = data.calls.successful || 0;
            if (failedCallsElement) failedCallsElement.textContent = data.calls.failed || 0;
        }

        if (data.registrations) {
            const totalRegElement = document.getElementById('totalRegistrations');
            const activeRegElement = document.getElementById('activeRegistrations');

            if (totalRegElement) totalRegElement.textContent = data.registrations.total || 0;
            if (activeRegElement) activeRegElement.textContent = data.registrations.active || 0;
        }

        // Update call rate chart if data is available
        if (data.call_rate && data.call_rate.length > 0) {
            this.updateLineChart('callRateChart', data.call_rate);
        }
    }

    updatePerformanceMetrics(data) {
        if (data.response_times) {
            const avgElement = document.getElementById('avgResponseTime');
            const p95Element = document.getElementById('p95ResponseTime');
            const p99Element = document.getElementById('p99ResponseTime');

            if (avgElement) avgElement.textContent = `${data.response_times.avg || 0}ms`;
            if (p95Element) p95Element.textContent = `${data.response_times.p95 || 0}ms`;
            if (p99Element) p99Element.textContent = `${data.response_times.p99 || 0}ms`;
        }

        if (data.throughput) {
            const rpsElement = document.getElementById('requestsPerSecond');
            if (rpsElement) rpsElement.textContent = `${data.throughput.requests_per_second || 0} req/s`;

            // Update throughput chart
            if (data.throughput.history && data.throughput.history.length > 0) {
                this.updateLineChart('throughputChart', data.throughput.history);
            }
        }

        if (data.errors) {
            const errorRateElement = document.getElementById('errorRate');
            const timeoutRateElement = document.getElementById('timeoutRate');

            if (errorRateElement) errorRateElement.textContent = `${(data.errors.error_rate || 0).toFixed(2)}%`;
            if (timeoutRateElement) timeoutRateElement.textContent = `${(data.errors.timeout_rate || 0).toFixed(2)}%`;
        }
    }

    updateRedisMetrics(data) {
        const connectionsElement = document.getElementById('redisConnections');
        const memoryElement = document.getElementById('redisMemoryUsage');
        const operationsElement = document.getElementById('redisOperations');
        const hitRateElement = document.getElementById('redisHitRate');

        if (connectionsElement) connectionsElement.textContent = data.connected_clients || 0;
        if (memoryElement) memoryElement.textContent = this.formatBytes(data.used_memory || 0);
        if (operationsElement) operationsElement.textContent = `${data.instantaneous_ops_per_sec || 0}/s`;
        if (hitRateElement) {
            const hitRate = data.keyspace_hits && data.keyspace_misses ? 
                (data.keyspace_hits / (data.keyspace_hits + data.keyspace_misses) * 100) : 0;
            hitRateElement.textContent = `${hitRate.toFixed(1)}%`;
        }
    }

    updateProgressBar(elementId, percentage) {
        const element = document.getElementById(elementId);
        if (!element) return;

        // Create progress bar if it doesn't exist
        if (!element.querySelector('.progress-bar')) {
            element.innerHTML = `
                <div class="progress-container">
                    <div class="progress-bar">
                        <div class="progress-fill"></div>
                    </div>
                </div>
            `;
        }

        const progressFill = element.querySelector('.progress-fill');
        if (progressFill) {
            progressFill.style.width = `${Math.min(100, Math.max(0, percentage))}%`;
            
            // Color coding
            if (percentage > 90) {
                progressFill.className = 'progress-fill progress-critical';
            } else if (percentage > 75) {
                progressFill.className = 'progress-fill progress-warning';
            } else {
                progressFill.className = 'progress-fill progress-normal';
            }
        }
    }

    updateLineChart(elementId, data) {
        const element = document.getElementById(elementId);
        if (!element || !data || data.length === 0) return;

        // Simple ASCII-style chart representation
        // In a real implementation, you'd use a charting library like Chart.js
        const maxValue = Math.max(...data.map(d => d.value || 0));
        const minValue = Math.min(...data.map(d => d.value || 0));
        const range = maxValue - minValue || 1;

        const chartHTML = `
            <div class="simple-chart">
                <div class="chart-header">
                    <span class="chart-max">${maxValue.toFixed(1)}</span>
                </div>
                <div class="chart-bars">
                    ${data.slice(-20).map(point => {
                        const height = ((point.value - minValue) / range) * 100;
                        return `<div class="chart-bar" style="height: ${height}%" title="${point.timestamp}: ${point.value}"></div>`;
                    }).join('')}
                </div>
                <div class="chart-footer">
                    <span class="chart-min">${minValue.toFixed(1)}</span>
                </div>
            </div>
        `;
        
        element.innerHTML = chartHTML;
    }

    loadTabData(tabId) {
        switch (tabId) {
            case 'system-metrics':
                this.loadSystemMetrics();
                break;
            case 'sip-metrics':
                this.loadSipMetrics();
                break;
            case 'performance-metrics':
                this.loadPerformanceMetrics();
                break;
            case 'redis-metrics':
                this.loadRedisMetrics();
                break;
        }
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    startAutoRefresh() {
        // Auto-refresh every 30 seconds
        this.refreshInterval = setInterval(() => {
            this.loadMetrics();
        }, 30000);
    }

    stopAutoRefresh() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }
}

// Initialize metrics manager when page loads
let metricsManager;

document.addEventListener('DOMContentLoaded', () => {
    if (document.getElementById('metrics')) {
        metricsManager = new MetricsManager();
    }
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (metricsManager) {
        metricsManager.stopAutoRefresh();
    }
});
