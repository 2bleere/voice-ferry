const express = require('express');
const router = express.Router();
const bcrypt = require('bcrypt');

// Use the shared auth service from app.locals instead of creating a new instance

// Login
router.post('/login', async (req, res) => {
  try {
    const { username, password } = req.body;
    
    if (!username || !password) {
      return res.status(400).json({ error: 'Username and password required' });
    }
    
    const result = await req.app.locals.authService.login(username, password);
    if (!result.success) {
      return res.status(401).json({ error: result.error });
    }
    
    res.json({
      success: true,
      token: result.token,
      user: result.user,
      expiresIn: result.expiresIn
    });
  } catch (error) {
    console.error('Login error:', error);
    res.status(500).json({ error: 'Login failed' });
  }
});

// Logout
router.post('/logout', async (req, res) => {
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');
    if (token) {
      await req.app.locals.authService.logout(token);
    }
    res.json({ success: true, message: 'Logged out successfully' });
  } catch (error) {
    console.error('Logout error:', error);
    res.status(500).json({ error: 'Logout failed' });
  }
});

// Refresh token
router.post('/refresh', async (req, res) => {
  try {
    const { refreshToken } = req.body;
    
    if (!refreshToken) {
      return res.status(400).json({ error: 'Refresh token required' });
    }
    
    const result = await req.app.locals.authService.refreshToken(refreshToken);
    if (!result.success) {
      return res.status(401).json({ error: result.error });
    }
    
    res.json({
      success: true,
      token: result.token,
      expiresIn: result.expiresIn
    });
  } catch (error) {
    console.error('Refresh token error:', error);
    res.status(500).json({ error: 'Token refresh failed' });
  }
});

// Get current user
router.get('/me', async (req, res) => {
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');
    if (!token) {
      return res.status(401).json({ error: 'No token provided' });
    }
    
    const user = await req.app.locals.authService.validateToken(token);
    res.json({ user });
  } catch (error) {
    res.status(401).json({ error: 'Invalid token' });
  }
});

// Change password
router.post('/change-password', async (req, res) => {
  try {
    const { currentPassword, newPassword } = req.body;
    const token = req.headers.authorization?.replace('Bearer ', '');
    
    if (!currentPassword || !newPassword) {
      return res.status(400).json({ error: 'Current and new passwords required' });
    }
    
    const user = await req.app.locals.authService.validateToken(token);
    const result = await req.app.locals.authService.changePassword(user.username, currentPassword, newPassword);
    
    if (!result.success) {
      return res.status(400).json({ error: result.error });
    }
    
    res.json({ success: true, message: 'Password changed successfully' });
  } catch (error) {
    console.error('Change password error:', error);
    res.status(500).json({ error: 'Password change failed' });
  }
});

// Get authentication settings
router.get('/settings', async (req, res) => {
  try {
    const settings = await req.app.locals.authService.getAuthSettings();
    res.json(settings);
  } catch (error) {
    console.error('Get auth settings error:', error);
    res.status(500).json({ error: 'Failed to get authentication settings' });
  }
});

// Update authentication settings (admin only)
router.put('/settings', async (req, res) => {
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');
    const user = await req.app.locals.authService.validateToken(token);
    
    if (user.role !== 'admin') {
      return res.status(403).json({ error: 'Admin access required' });
    }
    
    const settings = req.body;
    const result = await req.app.locals.authService.updateAuthSettings(settings);
    
    res.json(result);
  } catch (error) {
    console.error('Update auth settings error:', error);
    res.status(500).json({ error: 'Failed to update authentication settings' });
  }
});

module.exports = router;
