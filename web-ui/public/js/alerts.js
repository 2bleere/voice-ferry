// Alerts Page JavaScript
class AlertsManager {
    constructor() {
        this.currentSeverity = 'all';
        this.refreshInterval = null;
        this.init();
    }

    init() {
        this.attachEventListeners();
        this.loadAlerts();
        this.loadAlertRules();
        this.startAutoRefresh();
    }

    attachEventListeners() {
        // Severity filter
        const severitySelect = document.getElementById('alertSeverity');
        if (severitySelect) {
            severitySelect.addEventListener('change', (e) => {
                this.currentSeverity = e.target.value;
                this.loadAlerts();
            });
        }

        // Add alert rule button
        const addRuleBtn = document.getElementById('addAlertRule');
        if (addRuleBtn) {
            addRuleBtn.addEventListener('click', () => {
                this.showAlertRuleModal();
            });
        }

        // Refresh button
        const refreshBtn = document.getElementById('refreshAlerts');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.loadAlerts();
                this.loadAlertRules();
            });
        }

        // Alert rule modal events
        this.attachModalEventListeners();
    }

    attachModalEventListeners() {
        // Close modal events
        const closeModalBtn = document.getElementById('closeAlertRuleModal');
        const cancelBtn = document.getElementById('cancelAlertRule');
        
        if (closeModalBtn) {
            closeModalBtn.addEventListener('click', () => this.hideAlertRuleModal());
        }
        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => this.hideAlertRuleModal());
        }

        // Form submission
        const alertRuleForm = document.getElementById('alertRuleForm');
        if (alertRuleForm) {
            alertRuleForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.saveAlertRule();
            });
        }

        // Close modal when clicking outside
        const modal = document.getElementById('alertRuleModal');
        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.hideAlertRuleModal();
                }
            });
        }
    }

    async loadAlerts() {
        try {
            const params = new URLSearchParams();
            if (this.currentSeverity !== 'all') {
                params.append('severity', this.currentSeverity);
            }

            const response = await fetch(`/api/metrics/alerts/active?${params}`);
            if (!response.ok) throw new Error('Failed to fetch active alerts');
            
            const alerts = await response.json();
            this.displayActiveAlerts(alerts);
        } catch (error) {
            console.error('Failed to load alerts:', error);
            showToast('Failed to load alerts', 'error');
        }
    }

    async loadAlertRules() {
        try {
            const response = await fetch('/api/metrics/alerts/rules');
            if (!response.ok) throw new Error('Failed to fetch alert rules');
            
            const rules = await response.json();
            this.displayAlertRules(rules);
        } catch (error) {
            console.error('Failed to load alert rules:', error);
            showToast('Failed to load alert rules', 'error');
        }
    }

    displayActiveAlerts(alerts) {
        const alertsList = document.getElementById('activeAlertsList');
        const alertsCount = document.getElementById('activeAlertsCount');
        
        if (alertsCount) {
            alertsCount.textContent = Array.isArray(alerts) ? alerts.length : 0;
        }

        if (!alertsList) return;

        if (!Array.isArray(alerts) || alerts.length === 0) {
            alertsList.innerHTML = '<div class="alert-item alert-info">No active alerts</div>';
            return;
        }

        alertsList.innerHTML = alerts.map(alert => this.createAlertHTML(alert)).join('');
    }

    createAlertHTML(alert) {
        const timestamp = alert.timestamp ? 
            new Date(alert.timestamp).toLocaleString() : 
            'Unknown time';

        const severityClass = this.getSeverityClass(alert.severity);
        const severityIcon = this.getSeverityIcon(alert.severity);

        return `
            <div class="alert-item alert-${severityClass}" data-alert-id="${alert.id}">
                <div class="alert-header">
                    <div class="alert-severity">
                        <i class="${severityIcon}"></i>
                        <span class="alert-severity-text">${(alert.severity || 'info').toUpperCase()}</span>
                    </div>
                    <div class="alert-actions">
                        ${!alert.acknowledged ? `
                            <button class="btn btn-sm btn-secondary" onclick="alertsManager.acknowledgeAlert('${alert.id}')">
                                <i class="fas fa-check"></i> Acknowledge
                            </button>
                        ` : `
                            <span class="alert-acknowledged">
                                <i class="fas fa-check-circle"></i> Acknowledged
                            </span>
                        `}
                    </div>
                </div>
                <div class="alert-content">
                    <div class="alert-name">${alert.rule_name || 'Unknown Alert'}</div>
                    <div class="alert-message">${alert.message || 'No message'}</div>
                    <div class="alert-details">
                        <span class="alert-metric">Metric: ${alert.metric || 'Unknown'}</span>
                        <span class="alert-value">Value: ${alert.current_value || 'N/A'}</span>
                        <span class="alert-threshold">Threshold: ${alert.threshold || 'N/A'}</span>
                    </div>
                    <div class="alert-timestamp">
                        <i class="fas fa-clock"></i> ${timestamp}
                    </div>
                </div>
            </div>
        `;
    }

    displayAlertRules(rules) {
        const tableBody = document.querySelector('#alertRulesTable tbody');
        if (!tableBody) return;

        if (!Array.isArray(rules) || rules.length === 0) {
            tableBody.innerHTML = '<tr><td colspan="6" class="text-center">No alert rules configured</td></tr>';
            return;
        }

        tableBody.innerHTML = rules.map(rule => this.createRuleRowHTML(rule)).join('');
    }

    createRuleRowHTML(rule) {
        const severityClass = this.getSeverityClass(rule.severity);
        const statusClass = rule.enabled ? 'status-healthy' : 'status-inactive';
        const statusText = rule.enabled ? 'Enabled' : 'Disabled';

        return `
            <tr data-rule-id="${rule.id}">
                <td>${rule.name || 'Unnamed Rule'}</td>
                <td>${rule.metric || 'Unknown'} ${rule.condition || ''} ${rule.threshold || ''}</td>
                <td>
                    <span class="badge badge-${severityClass}">${(rule.severity || 'info').toUpperCase()}</span>
                </td>
                <td>${rule.threshold || 'N/A'}</td>
                <td>
                    <span class="status-indicator">
                        <span class="status-dot ${statusClass}"></span>
                        ${statusText}
                    </span>
                </td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-sm btn-secondary" onclick="alertsManager.editAlertRule('${rule.id}')">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="alertsManager.deleteAlertRule('${rule.id}')">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }

    getSeverityClass(severity) {
        switch (severity?.toLowerCase()) {
            case 'critical': return 'critical';
            case 'warning': return 'warning';
            case 'info': return 'info';
            default: return 'info';
        }
    }

    getSeverityIcon(severity) {
        switch (severity?.toLowerCase()) {
            case 'critical': return 'fas fa-exclamation-circle';
            case 'warning': return 'fas fa-exclamation-triangle';
            case 'info': return 'fas fa-info-circle';
            default: return 'fas fa-info-circle';
        }
    }

    async acknowledgeAlert(alertId) {
        try {
            const reason = prompt('Enter acknowledgment reason (optional):') || 'Acknowledged via web interface';
            
            const response = await fetch(`/api/metrics/alerts/${alertId}/acknowledge`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ reason })
            });

            if (!response.ok) throw new Error('Failed to acknowledge alert');
            
            showToast('Alert acknowledged', 'success');
            this.loadAlerts(); // Refresh alerts list
        } catch (error) {
            console.error('Failed to acknowledge alert:', error);
            showToast('Failed to acknowledge alert', 'error');
        }
    }

    showAlertRuleModal(rule = null) {
        const modal = document.getElementById('alertRuleModal');
        const modalTitle = document.getElementById('alertRuleModalTitle');
        const form = document.getElementById('alertRuleForm');
        
        if (!modal || !form) return;

        // Reset form
        form.reset();
        
        if (rule) {
            // Edit mode
            modalTitle.textContent = 'Edit Alert Rule';
            this.populateRuleForm(rule);
        } else {
            // Add mode
            modalTitle.textContent = 'Add Alert Rule';
        }

        modal.classList.add('active');
    }

    hideAlertRuleModal() {
        const modal = document.getElementById('alertRuleModal');
        if (modal) {
            modal.classList.remove('active');
        }
    }

    populateRuleForm(rule) {
        const form = document.getElementById('alertRuleForm');
        if (!form) return;

        const fields = ['alertRuleName', 'alertMetric', 'alertCondition', 'alertThreshold', 
                       'alertSeverityLevel', 'alertDuration', 'alertDescription', 'alertEnabled'];
        
        fields.forEach(fieldId => {
            const element = document.getElementById(fieldId);
            if (element && rule) {
                const ruleKey = fieldId.replace('alert', '').toLowerCase();
                const ruleKeyAlt = fieldId.replace('alert', '').toLowerCase().replace('level', '');
                
                if (rule[ruleKey] !== undefined) {
                    if (element.type === 'checkbox') {
                        element.checked = rule[ruleKey];
                    } else {
                        element.value = rule[ruleKey];
                    }
                } else if (rule[ruleKeyAlt] !== undefined) {
                    if (element.type === 'checkbox') {
                        element.checked = rule[ruleKeyAlt];
                    } else {
                        element.value = rule[ruleKeyAlt];
                    }
                }
            }
        });
    }

    async saveAlertRule() {
        try {
            const form = document.getElementById('alertRuleForm');
            if (!form) return;

            const formData = new FormData(form);
            const ruleData = {
                name: formData.get('ruleName'),
                metric: formData.get('metric'),
                condition: formData.get('condition'),
                threshold: parseFloat(formData.get('threshold')),
                severity: formData.get('severity'),
                duration: parseInt(formData.get('duration')),
                description: formData.get('description'),
                enabled: formData.has('enabled')
            };

            const response = await fetch('/api/metrics/alerts/rules', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(ruleData)
            });

            if (!response.ok) throw new Error('Failed to save alert rule');
            
            showToast('Alert rule saved successfully', 'success');
            this.hideAlertRuleModal();
            this.loadAlertRules();
        } catch (error) {
            console.error('Failed to save alert rule:', error);
            showToast('Failed to save alert rule', 'error');
        }
    }

    async editAlertRule(ruleId) {
        try {
            // For now, just show the add modal
            // In a real implementation, you'd fetch the rule data first
            this.showAlertRuleModal();
            showToast('Edit functionality coming soon', 'info');
        } catch (error) {
            console.error('Failed to edit alert rule:', error);
            showToast('Failed to edit alert rule', 'error');
        }
    }

    async deleteAlertRule(ruleId) {
        if (!confirm('Are you sure you want to delete this alert rule?')) {
            return;
        }

        try {
            const response = await fetch(`/api/metrics/alerts/rules/${ruleId}`, {
                method: 'DELETE'
            });

            if (!response.ok) throw new Error('Failed to delete alert rule');
            
            showToast('Alert rule deleted successfully', 'success');
            this.loadAlertRules();
        } catch (error) {
            console.error('Failed to delete alert rule:', error);
            showToast('Failed to delete alert rule', 'error');
        }
    }

    startAutoRefresh() {
        // Auto-refresh active alerts every 30 seconds
        this.refreshInterval = setInterval(() => {
            this.loadAlerts();
        }, 30000);
    }

    stopAutoRefresh() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }

    // Cleanup method
    destroy() {
        this.stopAutoRefresh();
    }
}

// Initialize alerts manager when page loads
let alertsManager;

document.addEventListener('DOMContentLoaded', () => {
    if (document.getElementById('alerts')) {
        alertsManager = new AlertsManager();
    }
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (alertsManager) {
        alertsManager.destroy();
    }
});
