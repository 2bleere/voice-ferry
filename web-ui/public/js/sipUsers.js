// SIP Users Management JavaScript
class SipUsersManager {
    constructor(app) {
        this.app = app;
        this.currentUsers = [];
        this.editingUser = null;
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Add user button
        const addUserBtn = document.getElementById('addSipUser');
        if (addUserBtn) {
            addUserBtn.addEventListener('click', () => this.showAddUserModal());
        }

        // Refresh button
        const refreshBtn = document.getElementById('refreshSipUsers');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refreshUsers());
        }

        // Modal close buttons
        const closeModalBtn = document.getElementById('closeSipUserModal');
        const cancelBtn = document.getElementById('cancelSipUser');
        if (closeModalBtn) {
            closeModalBtn.addEventListener('click', () => this.closeModal());
        }
        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => this.closeModal());
        }

        // Form submission
        const userForm = document.getElementById('sipUserForm');
        if (userForm) {
            userForm.addEventListener('submit', (e) => this.handleFormSubmit(e));
        }

        // Search and filter
        const searchInput = document.getElementById('sipUserSearch');
        const filterSelect = document.getElementById('sipUserFilter');
        if (searchInput) {
            searchInput.addEventListener('input', () => this.filterUsers());
        }
        if (filterSelect) {
            filterSelect.addEventListener('change', () => this.filterUsers());
        }

        // Close modal when clicking outside
        const modal = document.getElementById('sipUserModal');
        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.closeModal();
                }
            });
        }

        // Event delegation for action buttons in the table
        const sipUsersTable = document.getElementById('sipUsersTable');
        if (sipUsersTable) {
            sipUsersTable.addEventListener('click', (e) => {
                const button = e.target.closest('button[data-action]');
                if (!button) return;

                const action = button.getAttribute('data-action');
                const username = button.getAttribute('data-username');

                switch (action) {
                    case 'edit':
                        this.editUser(username);
                        break;
                    case 'toggle':
                        this.toggleUser(username);
                        break;
                    case 'delete':
                        this.deleteUser(username);
                        break;
                }
            });
        }
    }

    async loadUsers() {
        try {
            const response = await this.app.apiCall('/api/sip-users');
            if (response.success) {
                this.currentUsers = response.users || [];
                this.populateUsersTable(this.currentUsers);
                await this.loadStatistics();
            } else {
                this.app.showToast('error', 'Error', 'Failed to load SIP users');
            }
        } catch (error) {
            console.error('Load SIP users error:', error);
            this.app.showToast('error', 'Error', 'Failed to load SIP users');
        }
    }

    async loadStatistics() {
        try {
            // Calculate statistics from current users
            const stats = {
                total: this.currentUsers.length,
                enabled: this.currentUsers.filter(u => u.enabled).length,
                disabled: this.currentUsers.filter(u => !u.enabled).length,
                realms: new Set(this.currentUsers.map(u => u.realm)).size
            };

            // Update statistics display
            this.app.updateElement('totalSipUsers', stats.total);
            this.app.updateElement('enabledSipUsers', stats.enabled);
            this.app.updateElement('disabledSipUsers', stats.disabled);
            this.app.updateElement('sipUserRealms', stats.realms);
        } catch (error) {
            console.error('Load SIP users statistics error:', error);
        }
    }

    populateUsersTable(users) {
        const tbody = document.querySelector('#sipUsersTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (users.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td colspan="6" class="text-center">
                    <em>No SIP users found</em>
                </td>
            `;
            tbody.appendChild(row);
            return;
        }

        users.forEach(user => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>
                    <strong>${this.escapeHtml(user.username)}</strong>
                </td>
                <td>${this.escapeHtml(user.realm)}</td>
                <td>
                    <span class="status-badge ${user.enabled ? 'status-enabled' : 'status-disabled'}">
                        <i class="fas ${user.enabled ? 'fa-check-circle' : 'fa-times-circle'}"></i>
                        ${user.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                </td>
                <td>${this.formatTimestamp(user.createdAt)}</td>
                <td>${this.formatTimestamp(user.updatedAt)}</td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-sm btn-secondary" data-action="edit" data-username="${user.username}" title="Edit User">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm ${user.enabled ? 'btn-warning' : 'btn-success'}" 
                                data-action="toggle" data-username="${user.username}" 
                                title="${user.enabled ? 'Disable' : 'Enable'} User">
                            <i class="fas ${user.enabled ? 'fa-pause' : 'fa-play'}"></i>
                        </button>
                        ${user.username !== '787' ? `
                            <button class="btn btn-sm btn-danger" data-action="delete" data-username="${user.username}" title="Delete User">
                                <i class="fas fa-trash"></i>
                            </button>
                        ` : ''}
                    </div>
                </td>
            `;
            tbody.appendChild(row);
        });
    }

    filterUsers() {
        const searchTerm = document.getElementById('sipUserSearch')?.value.toLowerCase() || '';
        const filterValue = document.getElementById('sipUserFilter')?.value || 'all';

        let filteredUsers = this.currentUsers;

        // Apply search filter
        if (searchTerm) {
            filteredUsers = filteredUsers.filter(user => 
                user.username.toLowerCase().includes(searchTerm) ||
                user.realm.toLowerCase().includes(searchTerm)
            );
        }

        // Apply status filter
        if (filterValue !== 'all') {
            filteredUsers = filteredUsers.filter(user => {
                if (filterValue === 'enabled') return user.enabled;
                if (filterValue === 'disabled') return !user.enabled;
                return true;
            });
        }

        this.populateUsersTable(filteredUsers);
    }

    showAddUserModal() {
        this.editingUser = null;
        document.getElementById('sipUserModalTitle').textContent = 'Add SIP User';
        document.getElementById('saveSipUser').innerHTML = '<i class="fas fa-save"></i> Save User';
        
        // Reset form
        document.getElementById('sipUserForm').reset();
        document.getElementById('sipUsername').disabled = false;
        document.getElementById('sipRealm').value = 'sip-b2bua.local';
        document.getElementById('sipEnabled').checked = true;
        
        this.showModal();
    }

    async editUser(username) {
        try {
            const response = await this.app.apiCall(`/api/sip-users/${username}`);
            if (response.success && response.user) {
                this.editingUser = username;
                document.getElementById('sipUserModalTitle').textContent = 'Edit SIP User';
                document.getElementById('saveSipUser').innerHTML = '<i class="fas fa-save"></i> Update User';
                
                // Populate form
                document.getElementById('sipUsername').value = response.user.username;
                document.getElementById('sipUsername').disabled = true; // Don't allow username changes
                document.getElementById('sipPassword').value = ''; // Don't show current password
                document.getElementById('sipRealm').value = response.user.realm;
                document.getElementById('sipEnabled').checked = response.user.enabled;
                
                this.showModal();
            } else {
                this.app.showToast('error', 'Error', 'Failed to load user details');
            }
        } catch (error) {
            console.error('Edit user error:', error);
            this.app.showToast('error', 'Error', 'Failed to load user details');
        }
    }

    async handleFormSubmit(e) {
        e.preventDefault();
        
        const formData = new FormData(e.target);
        const userData = {
            username: formData.get('username'),
            password: formData.get('password'),
            realm: formData.get('realm'),
            enabled: formData.has('enabled')
        };

        // Validation
        if (!userData.username || !userData.realm) {
            this.app.showToast('error', 'Validation Error', 'Username and realm are required');
            return;
        }

        if (!this.editingUser && !userData.password) {
            this.app.showToast('error', 'Validation Error', 'Password is required for new users');
            return;
        }

        try {
            let response;
            if (this.editingUser) {
                // Update existing user
                const updateData = {
                    realm: userData.realm,
                    enabled: userData.enabled
                };
                
                // Only include password if it was provided
                if (userData.password) {
                    updateData.password = userData.password;
                }
                
                response = await this.app.apiCall(`/api/sip-users/${this.editingUser}`, {
                    method: 'PUT',
                    body: JSON.stringify(updateData)
                });
            } else {
                // Create new user
                response = await this.app.apiCall('/api/sip-users', {
                    method: 'POST',
                    body: JSON.stringify(userData)
                });
            }

            if (response.success) {
                this.app.showToast('success', 'Success', 
                    this.editingUser ? 'User updated successfully' : 'User created successfully');
                this.closeModal();
                await this.loadUsers();
            } else {
                this.app.showToast('error', 'Error', response.error || 'Operation failed');
            }
        } catch (error) {
            console.error('Save user error:', error);
            this.app.showToast('error', 'Error', 'Failed to save user');
        }
    }

    async toggleUser(username) {
        try {
            const response = await this.app.apiCall(`/api/sip-users/${username}/toggle`, {
                method: 'POST'
            });

            if (response.success) {
                const action = response.user.enabled ? 'enabled' : 'disabled';
                this.app.showToast('success', 'Success', `User ${action} successfully`);
                await this.loadUsers();
            } else {
                this.app.showToast('error', 'Error', response.error || 'Failed to toggle user status');
            }
        } catch (error) {
            console.error('Toggle user error:', error);
            this.app.showToast('error', 'Error', 'Failed to toggle user status');
        }
    }

    async deleteUser(username) {
        if (!confirm(`Are you sure you want to delete the SIP user "${username}"? This action cannot be undone.`)) {
            return;
        }

        try {
            const response = await this.app.apiCall(`/api/sip-users/${username}`, {
                method: 'DELETE'
            });

            if (response.success) {
                this.app.showToast('success', 'Success', 'User deleted successfully');
                await this.loadUsers();
            } else {
                this.app.showToast('error', 'Error', response.error || 'Failed to delete user');
            }
        } catch (error) {
            console.error('Delete user error:', error);
            this.app.showToast('error', 'Error', 'Failed to delete user');
        }
    }

    showModal() {
        const modal = document.getElementById('sipUserModal');
        if (modal) {
            modal.classList.add('active');
            // Focus on first input
            setTimeout(() => {
                const firstInput = modal.querySelector('input:not([disabled])');
                if (firstInput) firstInput.focus();
            }, 100);
        }
    }

    closeModal() {
        const modal = document.getElementById('sipUserModal');
        if (modal) {
            modal.classList.remove('active');
        }
        this.editingUser = null;
    }

    async refreshUsers() {
        await this.loadUsers();
        this.app.showToast('info', 'Refreshed', 'SIP users data updated');
    }

    formatTimestamp(timestamp) {
        if (!timestamp) return 'N/A';
        try {
            return new Date(timestamp).toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        } catch (error) {
            return 'Invalid Date';
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize when DOM is loaded
let sipUsersManager;
document.addEventListener('DOMContentLoaded', () => {
    // This will be initialized by the main app
});
