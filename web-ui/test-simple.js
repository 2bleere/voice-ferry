const yaml = require('yaml');
const fs = require('fs');
const SessionLimitsService = require('./services/sessionLimitsService');

async function testRedis() {
  console.log('🔍 Testing Redis operations...');
  
  const configData = fs.readFileSync('./config/b2bua.yaml', 'utf8');
  const config = yaml.parse(configData);
  
  process.env.MOCK_SESSION_LIMITS = 'false';
  const service = new SessionLimitsService(config);
  
  // Wait for Redis connection
  await new Promise(resolve => setTimeout(resolve, 1500));
  
  console.log('✅ Service ready:', service.isReady());
  
  if (service.isReady()) {
    try {
      // Test operations
      const limits = await service.getAllUserLimits();
      console.log('📊 Current limits:', limits);
      
      await service.setUserLimit('testuser', 15);
      console.log('✅ Set testuser limit to 15');
      
      const userLimit = await service.getUserLimit('testuser');
      console.log('👤 testuser limit:', userLimit);
      
      await service.deleteUserLimit('testuser');
      console.log('🗑️ Deleted testuser limit');
      
      console.log('🎉 All operations successful!');
    } catch (error) {
      console.error('❌ Operation failed:', error);
    }
  }
  
  await service.close();
}

testRedis().catch(console.error);
