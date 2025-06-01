const jwt = require('jsonwebtoken');
const bcrypt = require('bcrypt');
const fs = require('fs').promises;
const path = require('path');

class AuthService {
  constructor() {
    this.jwtSecret = process.env.JWT_SECRET || 'your-secret-key-change-in-production';
    this.jwtExpiry = process.env.JWT_EXPIRY || '24h';
    this.usersFile = path.join(__dirname, '../data/users.json');
    this.sessionsFile = path.join(__dirname, '../data/sessions.json');
    this.users = new Map();
    this.sessions = new Map();
    this.loadUsers();
    this.loadSessions();
  }

  async loadUsers() {
    try {
      const data = await fs.readFile(this.usersFile, 'utf8');
      const users = JSON.parse(data);
      this.users = new Map(Object.entries(users));
    } catch (error) {
      // Create default admin user if file doesn't exist
      const defaultAdmin = {
        username: 'admin',
        password: await bcrypt.hash('admin123', 10),
        role: 'admin',
        createdAt: new Date().toISOString(),
        lastLogin: null
      };
      this.users.set('admin', defaultAdmin);
      await this.saveUsers();
    }
  }

  async saveUsers() {
    try {
      await fs.mkdir(path.dirname(this.usersFile), { recursive: true });
      const usersObj = Object.fromEntries(this.users);
      await fs.writeFile(this.usersFile, JSON.stringify(usersObj, null, 2));
    } catch (error) {
      console.error('Failed to save users:', error);
    }
  }

  async loadSessions() {
    try {
      const data = await fs.readFile(this.sessionsFile, 'utf8');
      const sessions = JSON.parse(data);
      this.sessions = new Map(Object.entries(sessions));
    } catch (error) {
      // File doesn't exist or is invalid, start with empty sessions
      this.sessions = new Map();
      await this.saveSessions();
    }
  }

  async saveSessions() {
    try {
      await fs.mkdir(path.dirname(this.sessionsFile), { recursive: true });
      const sessionsObj = Object.fromEntries(this.sessions);
      await fs.writeFile(this.sessionsFile, JSON.stringify(sessionsObj, null, 2));
    } catch (error) {
      console.error('Failed to save sessions:', error);
    }
  }

  async login(username, password) {
    try {
      const user = this.users.get(username);
      if (!user) {
        return { success: false, error: 'Invalid credentials' };
      }

      const isValid = await bcrypt.compare(password, user.password);
      if (!isValid) {
        return { success: false, error: 'Invalid credentials' };
      }

      // Update last login
      user.lastLogin = new Date().toISOString();
      await this.saveUsers();

      // Generate JWT token
      const token = jwt.sign(
        { 
          username: user.username, 
          role: user.role,
          loginTime: user.lastLogin
        },
        this.jwtSecret,
        { expiresIn: this.jwtExpiry }
      );

      // Store session
      this.sessions.set(token, {
        username: user.username,
        loginTime: user.lastLogin,
        lastActivity: new Date().toISOString()
      });

      // Save sessions to disk
      await this.saveSessions();

      return {
        success: true,
        token,
        user: {
          username: user.username,
          role: user.role,
          lastLogin: user.lastLogin
        },
        expiresIn: this.jwtExpiry
      };
    } catch (error) {
      console.error('Login error:', error);
      return { success: false, error: 'Login failed' };
    }
  }

  async logout(token) {
    try {
      this.sessions.delete(token);
      await this.saveSessions();
      return { success: true };
    } catch (error) {
      console.error('Logout error:', error);
      return { success: false, error: 'Logout failed' };
    }
  }

  async validateToken(token) {
    try {
      const decoded = jwt.verify(token, this.jwtSecret);
      const session = this.sessions.get(token);
      
      if (!session) {
        throw new Error('Session not found');
      }

      // Update last activity
      session.lastActivity = new Date().toISOString();
      
      // Save sessions with updated activity (async, no need to wait)
      this.saveSessions().catch(err => console.error('Failed to save sessions:', err));
      
      return {
        username: decoded.username,
        role: decoded.role,
        loginTime: decoded.loginTime
      };
    } catch (error) {
      throw new Error('Invalid token');
    }
  }

  async refreshToken(refreshToken) {
    try {
      const decoded = jwt.verify(refreshToken, this.jwtSecret);
      const user = this.users.get(decoded.username);
      
      if (!user) {
        return { success: false, error: 'User not found' };
      }

      const newToken = jwt.sign(
        { 
          username: user.username, 
          role: user.role,
          loginTime: new Date().toISOString()
        },
        this.jwtSecret,
        { expiresIn: this.jwtExpiry }
      );

      return {
        success: true,
        token: newToken,
        expiresIn: this.jwtExpiry
      };
    } catch (error) {
      return { success: false, error: 'Invalid refresh token' };
    }
  }

  async changePassword(username, currentPassword, newPassword) {
    try {
      const user = this.users.get(username);
      if (!user) {
        return { success: false, error: 'User not found' };
      }

      const isValid = await bcrypt.compare(currentPassword, user.password);
      if (!isValid) {
        return { success: false, error: 'Current password is incorrect' };
      }

      user.password = await bcrypt.hash(newPassword, 10);
      await this.saveUsers();

      return { success: true };
    } catch (error) {
      console.error('Change password error:', error);
      return { success: false, error: 'Password change failed' };
    }
  }

  async getAuthSettings() {
    return {
      jwtExpiry: this.jwtExpiry,
      sessionTimeout: '24h',
      maxSessions: 10,
      passwordPolicy: {
        minLength: 8,
        requireUppercase: false,
        requireLowercase: false,
        requireNumbers: false,
        requireSpecialChars: false
      }
    };
  }

  async updateAuthSettings(settings) {
    // In a real implementation, this would update system settings
    console.log('Auth settings updated:', settings);
    return { success: true, settings };
  }

  async createUser(userData) {
    try {
      if (this.users.has(userData.username)) {
        return { success: false, error: 'User already exists' };
      }

      const hashedPassword = await bcrypt.hash(userData.password, 10);
      const newUser = {
        username: userData.username,
        password: hashedPassword,
        role: userData.role || 'user',
        createdAt: new Date().toISOString(),
        lastLogin: null
      };

      this.users.set(userData.username, newUser);
      await this.saveUsers();

      return { success: true, user: { username: newUser.username, role: newUser.role } };
    } catch (error) {
      return { success: false, error: 'Failed to create user' };
    }
  }

  async deleteUser(username) {
    try {
      if (!this.users.has(username)) {
        return { success: false, error: 'User not found' };
      }

      this.users.delete(username);
      await this.saveUsers();

      return { success: true };
    } catch (error) {
      return { success: false, error: 'Failed to delete user' };
    }
  }

  getActiveSessions() {
    const activeSessions = [];
    for (const [token, session] of this.sessions.entries()) {
      activeSessions.push({
        username: session.username,
        loginTime: session.loginTime,
        lastActivity: session.lastActivity,
        tokenHash: token.substring(0, 8) + '...'
      });
    }
    return activeSessions;
  }
}

module.exports = { AuthService };
