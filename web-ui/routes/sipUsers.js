const express = require('express');
const router = express.Router();
const Joi = require('joi');
const axios = require('axios');

// Validation schema for SIP user
const sipUserSchema = Joi.object({
  username: Joi.string().alphanum().min(3).max(30).required(),
  password: Joi.string().min(6).max(50).required(),
  realm: Joi.string().pattern(/^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$/).min(1).max(253).required(),
  enabled: Joi.boolean().default(true)
});

const sipUserUpdateSchema = Joi.object({
  password: Joi.string().min(6).max(50).optional(),
  realm: Joi.string().pattern(/^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$/).min(1).max(253).optional(),
  enabled: Joi.boolean().optional()
}).min(1); // At least one field must be provided

// In-memory storage for SIP users (in production, this would be in Redis/database)
let sipUsers = new Map();

// Initialize with default test user
sipUsers.set('787', {
  username: '787',
  password: '12345',
  realm: 'sip-b2bua.local',
  enabled: true,
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString()
});

// GET /api/sip-users - List all SIP users
router.get('/', async (req, res) => {
  try {
    const users = Array.from(sipUsers.values()).map(user => ({
      username: user.username,
      realm: user.realm,
      enabled: user.enabled,
      createdAt: user.createdAt,
      updatedAt: user.updatedAt
      // Don't expose password in list
    }));
    
    res.json({
      success: true,
      users: users,
      total: users.length
    });
  } catch (error) {
    console.error('Error fetching SIP users:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to fetch SIP users'
    });
  }
});

// GET /api/sip-users/:username - Get specific SIP user
router.get('/:username', async (req, res) => {
  try {
    const { username } = req.params;
    const user = sipUsers.get(username);
    
    if (!user) {
      return res.status(404).json({
        success: false,
        error: 'SIP user not found'
      });
    }
    
    // Return user without password
    const { password, ...userInfo } = user;
    res.json({
      success: true,
      user: userInfo
    });
  } catch (error) {
    console.error('Error fetching SIP user:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to fetch SIP user'
    });
  }
});

// POST /api/sip-users - Create new SIP user
router.post('/', async (req, res) => {
  try {
    const { error, value } = sipUserSchema.validate(req.body);
    if (error) {
      return res.status(400).json({
        success: false,
        error: `Validation error: ${error.details[0].message}`
      });
    }
    
    const { username, password, realm, enabled } = value;
    
    // Check if user already exists
    if (sipUsers.has(username)) {
      return res.status(409).json({
        success: false,
        error: 'SIP user already exists'
      });
    }
    
    // Create new user
    const newUser = {
      username,
      password,
      realm,
      enabled,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
    
    sipUsers.set(username, newUser);
    
    // In a real implementation, we would also update the Go backend's DigestAuth
    // For now, we'll simulate this by logging the action
    console.log(`Created SIP user: ${username} with realm: ${realm}`);
    
    // Return user without password
    const { password: _, ...userResponse } = newUser;
    res.status(201).json({
      success: true,
      message: 'SIP user created successfully',
      user: userResponse
    });
  } catch (error) {
    console.error('Error creating SIP user:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to create SIP user'
    });
  }
});

// PUT /api/sip-users/:username - Update existing SIP user
router.put('/:username', async (req, res) => {
  try {
    const { username } = req.params;
    const { error, value } = sipUserUpdateSchema.validate(req.body);
    
    if (error) {
      return res.status(400).json({
        success: false,
        error: `Validation error: ${error.details[0].message}`
      });
    }
    
    // Check if user exists
    const existingUser = sipUsers.get(username);
    if (!existingUser) {
      return res.status(404).json({
        success: false,
        error: 'SIP user not found'
      });
    }
    
    // Update user
    const updatedUser = {
      ...existingUser,
      ...value,
      updatedAt: new Date().toISOString()
    };
    
    sipUsers.set(username, updatedUser);
    
    console.log(`Updated SIP user: ${username}`);
    
    // Return user without password
    const { password: _, ...userResponse } = updatedUser;
    res.json({
      success: true,
      message: 'SIP user updated successfully',
      user: userResponse
    });
  } catch (error) {
    console.error('Error updating SIP user:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to update SIP user'
    });
  }
});

// DELETE /api/sip-users/:username - Delete SIP user
router.delete('/:username', async (req, res) => {
  try {
    const { username } = req.params;
    
    // Check if user exists
    if (!sipUsers.has(username)) {
      return res.status(404).json({
        success: false,
        error: 'SIP user not found'
      });
    }
    
    // Don't allow deletion of default test user
    if (username === '787') {
      return res.status(403).json({
        success: false,
        error: 'Cannot delete default test user'
      });
    }
    
    sipUsers.delete(username);
    
    console.log(`Deleted SIP user: ${username}`);
    
    res.json({
      success: true,
      message: 'SIP user deleted successfully'
    });
  } catch (error) {
    console.error('Error deleting SIP user:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to delete SIP user'
    });
  }
});

// POST /api/sip-users/:username/toggle - Toggle user enabled status
router.post('/:username/toggle', async (req, res) => {
  try {
    const { username } = req.params;
    
    // Check if user exists
    const existingUser = sipUsers.get(username);
    if (!existingUser) {
      return res.status(404).json({
        success: false,
        error: 'SIP user not found'
      });
    }
    
    // Toggle enabled status
    const updatedUser = {
      ...existingUser,
      enabled: !existingUser.enabled,
      updatedAt: new Date().toISOString()
    };
    
    sipUsers.set(username, updatedUser);
    
    const action = updatedUser.enabled ? 'enabled' : 'disabled';
    console.log(`${action} SIP user: ${username}`);
    
    // Return user without password
    const { password: _, ...userResponse } = updatedUser;
    res.json({
      success: true,
      message: `SIP user ${action} successfully`,
      user: userResponse
    });
  } catch (error) {
    console.error('Error toggling SIP user:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to toggle SIP user status'
    });
  }
});

// GET /api/sip-users/stats - Get SIP users statistics
router.get('/stats', async (req, res) => {
  try {
    const totalUsers = sipUsers.size;
    const enabledUsers = Array.from(sipUsers.values()).filter(user => user.enabled).length;
    const disabledUsers = totalUsers - enabledUsers;
    
    // Group by realm
    const realmStats = {};
    sipUsers.forEach(user => {
      if (!realmStats[user.realm]) {
        realmStats[user.realm] = { total: 0, enabled: 0, disabled: 0 };
      }
      realmStats[user.realm].total++;
      if (user.enabled) {
        realmStats[user.realm].enabled++;
      } else {
        realmStats[user.realm].disabled++;
      }
    });
    
    res.json({
      success: true,
      stats: {
        total: totalUsers,
        enabled: enabledUsers,
        disabled: disabledUsers,
        byRealm: realmStats
      }
    });
  } catch (error) {
    console.error('Error fetching SIP user stats:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to fetch SIP user statistics'
    });
  }
});

module.exports = router;
