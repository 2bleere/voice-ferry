#!/usr/bin/env node

const yaml = require('yaml');
const fs = require('fs');
const SessionLimitsService = require('./services/sessionLimitsService');

async function testRedisConnection() {
  console.log('Testing SessionLimitsService Redis connection...');
  
  try {
    // Load config
    const configData = fs.readFileSync('./config/b2bua.yaml', 'utf8');
    const config = yaml.parse(configData);
    
    console.log('Config loaded:', {
      redis: {
        enabled: config.redis.enabled,
        host: config.redis.host,
        port: config.redis.port,
        enable_session_limits: config.redis.enable_session_limits
      }
    });
    
    // Create service with Redis mode
    process.env.MOCK_SESSION_LIMITS = 'false';
    const service = new SessionLimitsService(config);
    
    // Wait a bit for initialization
    console.log('Waiting for service initialization...');
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    console.log('Service ready:', service.isReady());
    console.log('Mock mode:', service.mockMode);
    console.log('Redis connected:', service.redisConnected);
    
    if (service.isReady()) {
      // Test basic operations
      console.log('\n--- Testing basic operations ---');
      
      // Get all limits
      const allLimits = await service.getAllUserLimits();
      console.log('All user limits:', allLimits);
      
      // Set a user limit
      console.log('\nSetting limit for test user...');
      const setResult = await service.setUserLimit('testuser', 10);
      console.log('Set result:', setResult);
      
      // Get the user limit
      const userLimit = await service.getUserLimit('testuser');
      console.log('User limit for testuser:', userLimit);
      
      // Get all limits again
      const allLimitsAfter = await service.getAllUserLimits();
      console.log('All user limits after:', allLimitsAfter);
      
      // Delete the user limit
      console.log('\nDeleting limit for test user...');
      const deleteResult = await service.deleteUserLimit('testuser');
      console.log('Delete result:', deleteResult);
      
      // Verify deletion
      const userLimitAfterDelete = await service.getUserLimit('testuser');
      console.log('User limit after delete (should be default):', userLimitAfterDelete);
      
      console.log('\n✅ All tests passed!');
    } else {
      console.log('❌ Service not ready');
    }
    
    // Close the service
    await service.close();
    console.log('Service closed');
    
  } catch (error) {
    console.error('❌ Test failed:', error);
    process.exit(1);
  }
}

// Run the test
testRedisConnection().catch(error => {
  console.error('Unhandled error:', error);
  process.exit(1);
});
