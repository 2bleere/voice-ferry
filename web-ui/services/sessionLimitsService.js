// Session Limits Service
// Handles user session limits management through GRPC or Redis

const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const redis = require('redis');
const { promisify } = require('util');

class SessionLimitsService {
  constructor(config) {
    this.config = config;
    this.sessionLimitsEnabled = config.redis && config.redis.enable_session_limits;
    this.grpcClient = null;
    this.redisClient = null;
    this.redisConnected = false;
    this.mockMode = process.env.MOCK_SESSION_LIMITS === 'true';
    this.mockData = new Map(); // In-memory storage for testing
    this.keyPrefix = 'voice-ferry-c4:user-limit:'; // Match B2BUA Redis key format
    this.initialized = false;
    
    if (!this.mockMode && config.redis && config.redis.enabled && config.redis.enable_session_limits) {
      this._initClients().catch(err => {
        console.error('Failed to initialize Redis client, falling back to mock mode:', err);
        this.mockMode = true;
        this.mockData.set('_default', config.redis?.max_sessions_per_user || 5);
      });
    } else {
      console.log('SessionLimitsService: Running in mock mode');
      // Initialize with some test data
      this.mockData.set('_default', config.redis?.max_sessions_per_user || 5);
      this.initialized = true;
    }
  }
  
  async _initClients() {
    try {
      // Initialize gRPC client if needed
      if (this.config.grpc && this.config.grpc.enabled) {
        const packageDefinition = protoLoader.loadSync(
          'proto/b2bua/session_limits.proto',
          {
            keepCase: true,
            longs: String,
            enums: String,
            defaults: true,
            oneofs: true
          }
        );
        
        const sessionProto = grpc.loadPackageDefinition(packageDefinition).b2bua;
        
        this.grpcClient = new sessionProto.SessionService(
          `${this.config.grpc.host}:${this.config.grpc.port}`,
          grpc.credentials.createInsecure()
        );
      }
      
      // Initialize Redis client
      if (this.config.redis && this.config.redis.enabled) {
        console.log('Initializing Redis client for session limits...');
        
        this.redisClient = redis.createClient({
          url: process.env.REDIS_URL || `redis://${this.config.redis.host}:${this.config.redis.port}`,
          password: this.config.redis.password || undefined,
          database: this.config.redis.database || 0
        });
        
        this.redisClient.on('error', (err) => {
          console.error('Redis client error:', err);
          this.redisConnected = false;
        });
        
        this.redisClient.on('connect', () => {
          console.log('Redis client connected for session limits');
          this.redisConnected = true;
        });
        
        this.redisClient.on('ready', () => {
          console.log('Redis client ready for session limits');
          this.redisConnected = true;
        });
        
        await this.redisClient.connect();
        this.initialized = true;
        console.log('SessionLimitsService: Redis connection established');
      }
    } catch (error) {
      console.error('Failed to initialize Redis client:', error);
      throw error;
    }
  }
  
  /**
   * Get session limits for all users
   */
  async getAllUserLimits() {
    if (this.mockMode) {
      const limits = {};
      for (const [key, value] of this.mockData.entries()) {
        limits[key] = value;
      }
      return limits;
    }
    
    if (this.grpcClient) {
      // Use gRPC API
      return new Promise((resolve, reject) => {
        this.grpcClient.getUserSessionLimits({}, (err, response) => {
          if (err) {
            reject(err);
          } else {
            resolve(response.limits);
          }
        });
      });
    } else if (this.redisClient && this.redisConnected) {
      // Use Redis directly
      const pattern = "voice-ferry-c4:user-limit:*";
      const keys = await this.redisClient.keys(pattern);
      
      const limits = {};
      const defaultLimit = this.config.redis.max_sessions_per_user;
      
      // Add the default limit
      limits._default = defaultLimit;
      
      // Add user-specific limits
      for (const key of keys) {
        const username = key.substring("voice-ferry-c4:user-limit:".length);
        const limitStr = await this.redisClient.get(key);
        limits[username] = parseInt(limitStr, 10);
      }
      
      return limits;
    }
    
    // Fallback to config
    return { 
      _default: this.config.redis?.max_sessions_per_user || 5,
      ...(this.config.redis?.user_session_limits || {})
    };
  }
  
  /**
   * Get session limit for a specific user
   */
  async getUserLimit(username) {
    if (this.mockMode) {
      return this.mockData.get(username) || this.mockData.get('_default') || 5;
    }
    
    if (this.grpcClient) {
      // Use gRPC API
      return new Promise((resolve, reject) => {
        this.grpcClient.getUserSessionLimit({ username }, (err, response) => {
          if (err) {
            reject(err);
          } else {
            resolve(response.limit);
          }
        });
      });
    } else if (this.redisClient && this.redisConnected) {
      // Use Redis directly
      const key = `voice-ferry-c4:user-limit:${username}`;
      const limitStr = await this.redisClient.get(key);
      
      if (limitStr === null) {
        return this.config.redis.max_sessions_per_user;
      }
      
      return parseInt(limitStr, 10);
    }
    
    // Fallback to config
    if (this.config.redis?.user_session_limits && 
        this.config.redis.user_session_limits[username] !== undefined) {
      return this.config.redis.user_session_limits[username];
    }
    
    return this.config.redis?.max_sessions_per_user || 5;
  }
  
  /**
   * Set session limit for a specific user
   */
  async setUserLimit(username, limit) {
    if (typeof limit !== 'number' || isNaN(limit)) {
      throw new Error('Limit must be a number');
    }
    
    if (this.mockMode) {
      this.mockData.set(username, limit);
      return { success: true, username, limit };
    }
    
    if (this.grpcClient) {
      // Use gRPC API
      return new Promise((resolve, reject) => {
        this.grpcClient.setUserSessionLimit({ username, limit }, (err, response) => {
          if (err) {
            reject(err);
          } else {
            resolve(response);
          }
        });
      });
    } else if (this.redisClient && this.redisConnected) {
      // Use Redis directly
      const key = `voice-ferry-c4:user-limit:${username}`;
      await this.redisClient.set(key, limit.toString());
      
      return { success: true, username, limit };
    }
    
    throw new Error('No available method to set user session limit');
  }
  
  /**
   * Delete session limit for a specific user (revert to default)
   */
  async deleteUserLimit(username) {
    if (this.mockMode) {
      this.mockData.delete(username);
      return { success: true, username };
    }
    
    if (this.grpcClient) {
      // Use gRPC API
      return new Promise((resolve, reject) => {
        this.grpcClient.deleteUserSessionLimit({ username }, (err, response) => {
          if (err) {
            reject(err);
          } else {
            resolve(response);
          }
        });
      });
    } else if (this.redisClient && this.redisConnected) {
      // Use Redis directly
      const key = `voice-ferry-c4:user-limit:${username}`;
      await this.redisClient.del(key);
      
      return { success: true, username };
    }
    
    throw new Error('No available method to delete user session limit');
  }
  
  /**
   * Get current session counts for all users
   */
  async getUserSessionCounts() {
    if (this.mockMode) {
      // Return mock data for testing
      return {
        'alice': 2,
        'bob': 1,
        'carol': 3
      };
    }
    
    if (this.grpcClient) {
      // Use gRPC API
      return new Promise((resolve, reject) => {
        this.grpcClient.getUserSessionCounts({}, (err, response) => {
          if (err) {
            reject(err);
          } else {
            resolve(response.counts);
          }
        });
      });
    } else if (this.redisClient && this.redisConnected) {
      // Use Redis directly
      const pattern = "voice-ferry-c4:user-sessions:*";
      const keys = await this.redisClient.keys(pattern);
      
      const counts = {};
      
      for (const key of keys) {
        const username = key.substring("voice-ferry-c4:user-sessions:".length);
        const members = await this.redisClient.sMembers(key);
        counts[username] = members.length;
      }
      
      return counts;
    }
    
    return {};
  }
  
  /**
   * Close clients when service is shutting down
   */
  async close() {
    if (this.redisClient && this.redisConnected) {
      try {
        await this.redisClient.quit();
        console.log('Redis client disconnected for session limits');
      } catch (error) {
        console.error('Error closing Redis client:', error);
      }
    }
  }
  
  /**
   * Check if the service is ready to handle requests
   */
  isReady() {
    return this.initialized && (this.mockMode || this.redisConnected);
  }
}

module.exports = SessionLimitsService;
