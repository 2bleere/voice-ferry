// Configuration Management JavaScript
class ConfigManager {
    constructor(app) {
        this.app = app;
        this.currentConfig = {};
        this.isDirty = false;
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Save configuration button
        const saveBtn = document.getElementById('saveConfig');
        if (saveBtn) {
            saveBtn.addEventListener('click', () => this.saveConfiguration());
        }

        // Backup configuration button
        const backupBtn = document.getElementById('backupConfig');
        if (backupBtn) {
            backupBtn.addEventListener('click', () => this.backupConfiguration());
        }

        // Restore configuration button
        const restoreBtn = document.getElementById('restoreConfig');
        if (restoreBtn) {
            restoreBtn.addEventListener('click', () => this.restoreConfiguration());
        }

        // Configuration tabs
        const tabButtons = document.querySelectorAll('.config-tab');
        tabButtons.forEach(button => {
            button.addEventListener('click', (e) => this.switchTab(e));
        });

        // Form change detection
        this.setupChangeDetection();
    }

    setupChangeDetection() {
        const configForm = document.getElementById('configForm');
        if (configForm) {
            configForm.addEventListener('input', () => {
                this.isDirty = true;
                this.updateSaveButton();
            });
        }
    }

    async loadConfiguration() {
        try {
            await Promise.all([
                this.loadSipConfig(),
                this.loadRedisConfig(),
                this.loadEtcdConfig(),
                this.loadLoggingConfig(),
                this.loadRoutingConfig()
            ]);
            
            this.isDirty = false;
            this.updateSaveButton();
        } catch (error) {
            console.error('Load configuration error:', error);
            this.app.showToast('error', 'Error', 'Failed to load configuration');
        }
    }

    async loadSipConfig() {
        try {
            const response = await this.app.apiCall('/api/config/sip');
            if (response && response.success !== false) {
                this.populateSipConfig(response);
            }
        } catch (error) {
            console.error('Load SIP config error:', error);
        }
    }

    async loadRedisConfig() {
        try {
            const response = await this.app.apiCall('/api/config/redis');
            if (response && response.success !== false) {
                this.populateRedisConfig(response);
            }
        } catch (error) {
            console.error('Load Redis config error:', error);
        }
    }

    async loadEtcdConfig() {
        try {
            const response = await this.app.apiCall('/api/config/etcd');
            if (response && response.success !== false) {
                this.populateEtcdConfig(response);
            }
        } catch (error) {
            console.error('Load etcd config error:', error);
        }
    }

    async loadLoggingConfig() {
        try {
            const response = await this.app.apiCall('/api/config/logging');
            if (response && response.success !== false) {
                this.populateLoggingConfig(response);
            }
        } catch (error) {
            console.error('Load logging config error:', error);
        }
    }

    async loadRoutingConfig() {
        try {
            const response = await this.app.apiCall('/api/config/routing');
            if (response && response.success !== false) {
                this.populateRoutingConfig(response);
            }
        } catch (error) {
            console.error('Load routing config error:', error);
        }
    }

    populateSipConfig(config) {
        this.app.setFormValue('sipHost', config.host);
        this.app.setFormValue('sipPort', config.port);
        this.app.setFormValue('sipTransport', config.transport);
        this.app.setFormValue('sipTransactionTimeout', config.timeouts?.transaction);
        this.app.setFormValue('sipDialogTimeout', config.timeouts?.dialog);
        this.app.setFormValue('sipRegistrationTimeout', config.timeouts?.registration);
        this.app.setFormValue('sipTlsEnabled', config.tls?.enabled);
        this.app.setFormValue('sipTlsCertFile', config.tls?.cert_file);
        this.app.setFormValue('sipTlsKeyFile', config.tls?.key_file);
        this.app.setFormValue('sipTlsCaFile', config.tls?.ca_file);
    }

    populateRedisConfig(config) {
        this.app.setFormValue('redisEnabled', config.enabled);
        this.app.setFormValue('redisHost', config.host);
        this.app.setFormValue('redisPort', config.port);
        this.app.setFormValue('redisPassword', config.password);
        this.app.setFormValue('redisDatabase', config.database);
        this.app.setFormValue('redisPoolSize', config.pool_size);
        this.app.setFormValue('redisMinIdleConns', config.min_idle_conns);
        this.app.setFormValue('redisMaxIdleTime', config.max_idle_time);
        this.app.setFormValue('redisConnMaxLifetime', config.conn_max_lifetime);
        this.app.setFormValue('redisTimeout', config.timeout);
        this.app.setFormValue('redisEnableSessionLimits', config.enable_session_limits);
        this.app.setFormValue('redisMaxSessionsPerUser', config.max_sessions_per_user);
        this.app.setFormValue('redisSessionLimitAction', config.session_limit_action);
    }

    populateEtcdConfig(config) {
        this.app.setFormValue('etcdEnabled', config.enabled);
        this.app.setFormValue('etcdEndpoints', config.endpoints?.join(','));
        this.app.setFormValue('etcdTimeout', config.timeout);
        this.app.setFormValue('etcdUsername', config.username);
        this.app.setFormValue('etcdPassword', config.password);
        this.app.setFormValue('etcdPrefix', config.prefix);
        this.app.setFormValue('etcdAutoSyncInterval', config.auto_sync_interval);
        this.app.setFormValue('etcdDialTimeout', config.dial_timeout);
        this.app.setFormValue('etcdDialKeepAliveTime', config.dial_keep_alive_time);
        this.app.setFormValue('etcdDialKeepAliveTimeout', config.dial_keep_alive_timeout);
    }

    populateLoggingConfig(config) {
        this.app.setFormValue('logLevel', config.level);
        this.app.setFormValue('logFormat', config.format);
        this.app.setFormValue('logOutput', config.output);
        this.app.setFormValue('logFile', config.file);
        this.app.setFormValue('logMaxSize', config.max_size);
        this.app.setFormValue('logMaxBackups', config.max_backups);
        this.app.setFormValue('logMaxAge', config.max_age);
    }

    populateRoutingConfig(config) {
        this.app.setFormValue('routingDefaultAction', config.default_action);
        this.app.setFormValue('routingDefaultDestination', config.default_destination);
        this.app.setFormValue('routingEnableFailover', config.enable_failover);
        this.app.setFormValue('routingFailoverTimeout', config.failover_timeout);
    }

    async saveConfiguration() {
        if (!this.isDirty) {
            this.app.showToast('info', 'Info', 'No changes to save');
            return;
        }

        const saveBtn = document.getElementById('saveConfig');
        if (saveBtn) {
            saveBtn.disabled = true;
            saveBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Saving...';
        }

        try {
            const config = this.collectConfigData();
            
            await Promise.all([
                this.saveSipConfig(config.sip),
                this.saveRedisConfig(config.redis),
                this.saveEtcdConfig(config.etcd),
                this.saveLoggingConfig(config.logging),
                this.saveRoutingConfig(config.routing)
            ]);

            this.isDirty = false;
            this.updateSaveButton();
            this.app.showToast('success', 'Success', 'Configuration saved successfully');
        } catch (error) {
            console.error('Save configuration error:', error);
            this.app.showToast('error', 'Error', 'Failed to save configuration');
        } finally {
            if (saveBtn) {
                saveBtn.disabled = false;
                saveBtn.innerHTML = '<i class="fas fa-save"></i> Save Configuration';
            }
        }
    }

    collectConfigData() {
        return {
            sip: {
                host: this.app.getFormValue('sipHost'),
                port: parseInt(this.app.getFormValue('sipPort')),
                transport: this.app.getFormValue('sipTransport'),
                timeouts: {
                    transaction: this.app.getFormValue('sipTransactionTimeout'),
                    dialog: this.app.getFormValue('sipDialogTimeout'),
                    registration: this.app.getFormValue('sipRegistrationTimeout')
                },
                tls: {
                    enabled: this.app.getFormValue('sipTlsEnabled'),
                    cert_file: this.app.getFormValue('sipTlsCertFile'),
                    key_file: this.app.getFormValue('sipTlsKeyFile'),
                    ca_file: this.app.getFormValue('sipTlsCaFile')
                }
            },
            redis: {
                enabled: this.app.getFormValue('redisEnabled'),
                host: this.app.getFormValue('redisHost'),
                port: parseInt(this.app.getFormValue('redisPort')),
                password: this.app.getFormValue('redisPassword'),
                database: parseInt(this.app.getFormValue('redisDatabase')),
                pool_size: parseInt(this.app.getFormValue('redisPoolSize')),
                min_idle_conns: parseInt(this.app.getFormValue('redisMinIdleConns')),
                max_idle_time: parseInt(this.app.getFormValue('redisMaxIdleTime')),
                conn_max_lifetime: parseInt(this.app.getFormValue('redisConnMaxLifetime')),
                timeout: parseInt(this.app.getFormValue('redisTimeout')),
                enable_session_limits: this.app.getFormValue('redisEnableSessionLimits'),
                max_sessions_per_user: parseInt(this.app.getFormValue('redisMaxSessionsPerUser')),
                session_limit_action: this.app.getFormValue('redisSessionLimitAction')
            },
            etcd: {
                enabled: this.app.getFormValue('etcdEnabled'),
                endpoints: this.app.getFormValue('etcdEndpoints').split(',').map(s => s.trim()),
                timeout: parseInt(this.app.getFormValue('etcdTimeout')),
                username: this.app.getFormValue('etcdUsername'),
                password: this.app.getFormValue('etcdPassword'),
                prefix: this.app.getFormValue('etcdPrefix'),
                auto_sync_interval: parseInt(this.app.getFormValue('etcdAutoSyncInterval')),
                dial_timeout: parseInt(this.app.getFormValue('etcdDialTimeout')),
                dial_keep_alive_time: parseInt(this.app.getFormValue('etcdDialKeepAliveTime')),
                dial_keep_alive_timeout: parseInt(this.app.getFormValue('etcdDialKeepAliveTimeout'))
            },
            logging: {
                level: this.app.getFormValue('logLevel'),
                format: this.app.getFormValue('logFormat'),
                output: this.app.getFormValue('logOutput'),
                file: this.app.getFormValue('logFile'),
                max_size: parseInt(this.app.getFormValue('logMaxSize')),
                max_backups: parseInt(this.app.getFormValue('logMaxBackups')),
                max_age: parseInt(this.app.getFormValue('logMaxAge'))
            },
            routing: {
                default_action: this.app.getFormValue('routingDefaultAction'),
                default_destination: this.app.getFormValue('routingDefaultDestination'),
                enable_failover: this.app.getFormValue('routingEnableFailover'),
                failover_timeout: parseInt(this.app.getFormValue('routingFailoverTimeout'))
            }
        };
    }

    async saveSipConfig(config) {
        const response = await this.app.apiCall('/api/config/sip', {
            method: 'PUT',
            body: JSON.stringify(config)
        });
        
        if (response && response.success === false) {
            throw new Error(response.error || 'Failed to save SIP configuration');
        }
    }

    async saveRedisConfig(config) {
        const response = await this.app.apiCall('/api/config/redis', {
            method: 'PUT',
            body: JSON.stringify(config)
        });
        
        if (response && response.success === false) {
            throw new Error(response.error || 'Failed to save Redis configuration');
        }
    }

    async saveEtcdConfig(config) {
        const response = await this.app.apiCall('/api/config/etcd', {
            method: 'PUT',
            body: JSON.stringify(config)
        });
        
        if (response && response.success === false) {
            throw new Error(response.error || 'Failed to save etcd configuration');
        }
    }

    async saveLoggingConfig(config) {
        const response = await this.app.apiCall('/api/config/logging', {
            method: 'PUT',
            body: JSON.stringify(config)
        });
        
        if (response && response.success === false) {
            throw new Error(response.error || 'Failed to save logging configuration');
        }
    }

    async saveRoutingConfig(config) {
        const response = await this.app.apiCall('/api/config/routing', {
            method: 'PUT',
            body: JSON.stringify(config)
        });
        
        if (response && response.success === false) {
            throw new Error(response.error || 'Failed to save routing configuration');
        }
    }

    async backupConfiguration() {
        try {
            const response = await this.app.apiCall('/api/config/backup', {
                method: 'POST'
            });

            if (response && response.success) {
                // Create download link for backup file
                const blob = new Blob([JSON.stringify(response.config, null, 2)], {
                    type: 'application/json'
                });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `voice-ferry-config-backup-${new Date().toISOString().split('T')[0]}.json`;
                a.click();
                URL.revokeObjectURL(url);

                this.app.showToast('success', 'Success', 'Configuration backup created');
            } else {
                throw new Error(response?.error || 'Failed to create backup');
            }
        } catch (error) {
            console.error('Backup configuration error:', error);
            this.app.showToast('error', 'Error', 'Failed to create configuration backup');
        }
    }

    async restoreConfiguration() {
        const fileInput = document.createElement('input');
        fileInput.type = 'file';
        fileInput.accept = '.json';
        
        fileInput.addEventListener('change', async (e) => {
            const file = e.target.files[0];
            if (!file) return;

            try {
                const text = await file.text();
                const config = JSON.parse(text);

                const response = await this.app.apiCall('/api/config/restore', {
                    method: 'POST',
                    body: JSON.stringify(config)
                });

                if (response && response.success) {
                    await this.loadConfiguration();
                    this.app.showToast('success', 'Success', 'Configuration restored successfully');
                } else {
                    throw new Error(response?.error || 'Failed to restore configuration');
                }
            } catch (error) {
                console.error('Restore configuration error:', error);
                this.app.showToast('error', 'Error', 'Failed to restore configuration');
            }
        });

        fileInput.click();
    }

    switchTab(e) {
        e.preventDefault();
        
        const tabId = e.target.getAttribute('data-tab');
        
        // Update active tab
        document.querySelectorAll('.config-tab').forEach(tab => {
            tab.classList.remove('active');
        });
        e.target.classList.add('active');
        
        // Show corresponding tab content
        document.querySelectorAll('.config-section').forEach(section => {
            section.classList.remove('active');
        });
        
        const targetSection = document.getElementById(`${tabId}Config`);
        if (targetSection) {
            targetSection.classList.add('active');
        }
    }

    updateSaveButton() {
        const saveBtn = document.getElementById('saveConfig');
        if (saveBtn) {
            if (this.isDirty) {
                saveBtn.disabled = false;
                saveBtn.classList.add('btn-warning');
                saveBtn.innerHTML = '<i class="fas fa-save"></i> Save Changes';
            } else {
                saveBtn.disabled = true;
                saveBtn.classList.remove('btn-warning');
                saveBtn.innerHTML = '<i class="fas fa-save"></i> Save Configuration';
            }
        }
    }

    async testConnection(service) {
        try {
            const response = await this.app.apiCall(`/api/config/test/${service}`, {
                method: 'POST'
            });

            if (response && response.success) {
                this.app.showToast('success', 'Success', `${service} connection test successful`);
            } else {
                this.app.showToast('error', 'Error', response?.error || `${service} connection test failed`);
            }
        } catch (error) {
            console.error(`Test ${service} connection error:`, error);
            this.app.showToast('error', 'Error', `Failed to test ${service} connection`);
        }
    }
}

// Initialize config manager when DOM is loaded
let configManager;
document.addEventListener('DOMContentLoaded', () => {
    if (window.app) {
        configManager = new ConfigManager(window.app);
    }
});
