// Logs Page JavaScript
class LogsManager {
    constructor() {
        this.isStreaming = false;
        this.streamEventSource = null;
        this.currentLevel = 'info';
        this.currentLines = 500;
        this.maxLogLines = 1000;
        this.init();
    }

    init() {
        this.attachEventListeners();
        this.loadLogs();
    }

    attachEventListeners() {
        // Log level selector
        const logLevelSelect = document.getElementById('logLevel');
        if (logLevelSelect) {
            logLevelSelect.addEventListener('change', (e) => {
                this.currentLevel = e.target.value === 'all' ? null : e.target.value;
                if (this.isStreaming) {
                    this.stopStreaming();
                    this.startStreaming();
                } else {
                    this.loadLogs();
                }
            });
        }

        // Log lines selector
        const logLinesSelect = document.getElementById('logLines');
        if (logLinesSelect) {
            logLinesSelect.addEventListener('change', (e) => {
                this.currentLines = parseInt(e.target.value);
                if (!this.isStreaming) {
                    this.loadLogs();
                }
            });
        }

        // Clear logs button
        const clearBtn = document.getElementById('clearLogs');
        if (clearBtn) {
            clearBtn.addEventListener('click', () => {
                this.clearLogs();
            });
        }

        // Refresh logs button
        const refreshBtn = document.getElementById('refreshLogs');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.loadLogs();
            });
        }

        // Stream logs button
        const streamBtn = document.getElementById('streamLogs');
        if (streamBtn) {
            streamBtn.addEventListener('click', () => {
                if (this.isStreaming) {
                    this.stopStreaming();
                } else {
                    this.startStreaming();
                }
            });
        }
    }

    async loadLogs() {
        try {
            const params = new URLSearchParams({
                lines: this.currentLines.toString()
            });
            
            if (this.currentLevel) {
                params.append('level', this.currentLevel);
            }

            const response = await fetch(`/api/logs?${params}`);
            if (!response.ok) throw new Error('Failed to fetch logs');
            
            const logs = await response.json();
            this.displayLogs(logs);
        } catch (error) {
            console.error('Failed to load logs:', error);
            showToast('Failed to load logs', 'error');
        }
    }

    startStreaming() {
        if (this.isStreaming) return;

        try {
            const params = new URLSearchParams({
                stream: 'true'
            });
            
            if (this.currentLevel) {
                params.append('level', this.currentLevel);
            }

            this.streamEventSource = new EventSource(`/api/logs?${params}`);
            
            this.streamEventSource.onopen = () => {
                this.isStreaming = true;
                this.updateStreamStatus();
                showToast('Log streaming started', 'success');
            };

            this.streamEventSource.onmessage = (event) => {
                try {
                    const logEntry = JSON.parse(event.data);
                    this.appendLogEntry(logEntry);
                } catch (error) {
                    console.error('Error parsing log entry:', error);
                }
            };

            this.streamEventSource.onerror = (error) => {
                console.error('Log stream error:', error);
                this.stopStreaming();
                showToast('Log streaming error', 'error');
            };

        } catch (error) {
            console.error('Failed to start log streaming:', error);
            showToast('Failed to start log streaming', 'error');
        }
    }

    stopStreaming() {
        if (!this.isStreaming) return;

        if (this.streamEventSource) {
            this.streamEventSource.close();
            this.streamEventSource = null;
        }

        this.isStreaming = false;
        this.updateStreamStatus();
        showToast('Log streaming stopped', 'info');
    }

    updateStreamStatus() {
        const statusElement = document.getElementById('logStatus');
        const streamBtn = document.getElementById('streamLogs');
        
        if (statusElement) {
            const statusDot = statusElement.querySelector('.status-dot');
            const statusText = statusElement.querySelector('.status-text');
            
            if (this.isStreaming) {
                statusDot.className = 'status-dot status-healthy';
                statusText.textContent = 'Streaming';
            } else {
                statusDot.className = 'status-dot status-inactive';
                statusText.textContent = 'Not Streaming';
            }
        }

        if (streamBtn) {
            const icon = streamBtn.querySelector('i');
            if (this.isStreaming) {
                streamBtn.innerHTML = '<i class="fas fa-stop"></i> Stop Stream';
                streamBtn.classList.remove('btn-primary');
                streamBtn.classList.add('btn-warning');
            } else {
                streamBtn.innerHTML = '<i class="fas fa-play"></i> Start Stream';
                streamBtn.classList.remove('btn-warning');
                streamBtn.classList.add('btn-primary');
            }
        }
    }

    displayLogs(logs) {
        const logOutput = document.getElementById('logOutput');
        if (!logOutput) return;

        if (!Array.isArray(logs)) {
            logOutput.innerHTML = '<div class="log-entry log-error">No logs available</div>';
            return;
        }

        logOutput.innerHTML = '';
        logs.forEach(log => {
            this.appendLogEntry(log);
        });

        // Auto-scroll to bottom
        logOutput.scrollTop = logOutput.scrollHeight;
    }

    appendLogEntry(logEntry) {
        const logOutput = document.getElementById('logOutput');
        if (!logOutput) return;

        const logElement = this.createLogElement(logEntry);
        logOutput.appendChild(logElement);

        // Remove old entries if we exceed max lines
        const logEntries = logOutput.querySelectorAll('.log-entry');
        if (logEntries.length > this.maxLogLines) {
            const entriesToRemove = logEntries.length - this.maxLogLines;
            for (let i = 0; i < entriesToRemove; i++) {
                logEntries[i].remove();
            }
        }

        // Auto-scroll to bottom if user is at bottom
        const isAtBottom = logOutput.scrollTop + logOutput.clientHeight >= logOutput.scrollHeight - 100;
        if (isAtBottom) {
            logOutput.scrollTop = logOutput.scrollHeight;
        }
    }

    createLogElement(logEntry) {
        const logDiv = document.createElement('div');
        logDiv.className = `log-entry log-${logEntry.level || 'info'}`;

        const timestamp = logEntry.timestamp ? 
            new Date(logEntry.timestamp).toLocaleString() : 
            new Date().toLocaleString();

        const level = (logEntry.level || 'INFO').toUpperCase();
        const message = logEntry.message || logEntry.msg || 'No message';
        const component = logEntry.component || logEntry.module || '';

        logDiv.innerHTML = `
            <span class="log-timestamp">${timestamp}</span>
            <span class="log-level log-level-${logEntry.level || 'info'}">${level}</span>
            ${component ? `<span class="log-component">[${component}]</span>` : ''}
            <span class="log-message">${this.escapeHtml(message)}</span>
        `;

        // Add extra fields if present
        if (logEntry.fields && Object.keys(logEntry.fields).length > 0) {
            const fieldsDiv = document.createElement('div');
            fieldsDiv.className = 'log-fields';
            fieldsDiv.innerHTML = Object.entries(logEntry.fields)
                .map(([key, value]) => `<span class="log-field">${key}: ${this.escapeHtml(String(value))}</span>`)
                .join(' ');
            logDiv.appendChild(fieldsDiv);
        }

        return logDiv;
    }

    clearLogs() {
        const logOutput = document.getElementById('logOutput');
        if (logOutput) {
            logOutput.innerHTML = '';
            showToast('Logs cleared', 'info');
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Cleanup method
    destroy() {
        this.stopStreaming();
    }
}

// Initialize logs manager when page loads
let logsManager;

document.addEventListener('DOMContentLoaded', () => {
    if (document.getElementById('logs')) {
        logsManager = new LogsManager();
    }
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (logsManager) {
        logsManager.destroy();
    }
});
