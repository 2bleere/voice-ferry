const express = require('express');
const http = require('http');
const WebSocket = require('ws');
const path = require('path');
const helmet = require('helmet');
const cors = require('cors');
const rateLimit = require('express-rate-limit');
const compression = require('compression');
const morgan = require('morgan');

const configRoutes = require('./routes/config');
const dashboardRoutes = require('./routes/dashboard');
const apiRoutes = require('./routes/api');
const authRoutes = require('./routes/auth');
const sessionRoutes = require('./routes/sessions');
const metricsRoutes = require('./routes/metrics');
const sipUsersRoutes = require('./routes/sipUsers');

const { WebSocketManager } = require('./services/websocket');
const { ConfigManager } = require('./services/config');
const { MonitoringService } = require('./services/monitoring');
const { AuthService } = require('./services/auth');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

// Add health check endpoint early (before rate limiting)
app.get('/api/health', (req, res) => {
  const health = {
    status: 'ok',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    environment: process.env.NODE_ENV || 'development',
    version: require('./package.json').version
  };
  
  res.status(200).json(health);
});

// Middleware
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://cdnjs.cloudflare.com"],
      scriptSrc: ["'self'", "https://cdnjs.cloudflare.com", "https://cdn.jsdelivr.net"],
      fontSrc: ["'self'", "https://cdnjs.cloudflare.com"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", "ws:", "wss:"]
    }
  }
}));

app.use(compression());
app.use(cors());
app.use(morgan('combined'));

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 1000 // limit each IP to 1000 requests per windowMs
});
app.use('/api/', limiter);

// Body parsing
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

// Static files
app.use(express.static(path.join(__dirname, 'public')));

// Services
const authService = new AuthService();
const configManager = new ConfigManager();
const monitoringService = new MonitoringService();
const wsManager = new WebSocketManager(wss, monitoringService);

// Make services available to routes
app.locals.authService = authService;
app.locals.configManager = configManager;
app.locals.monitoringService = monitoringService;
app.locals.wsManager = wsManager;

// Authentication middleware
const authenticate = async (req, res, next) => {
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');
    if (!token) {
      return res.status(401).json({ error: 'No token provided' });
    }
    
    const user = await authService.validateToken(token);
    req.user = user;
    next();
  } catch (error) {
    res.status(401).json({ error: 'Invalid token' });
  }
};

// Routes
app.use('/api/auth', authRoutes);
app.use('/api/config', authenticate, configRoutes);
app.use('/api/dashboard', authenticate, dashboardRoutes);
app.use('/api/sessions', authenticate, sessionRoutes);
app.use('/api/metrics', authenticate, metricsRoutes);
app.use('/api/sip-users', authenticate, sipUsersRoutes);
app.use('/api', authenticate, apiRoutes);

// Serve main application
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

// Health check
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    version: process.env.VERSION || '1.0.0'
  });
});

// Error handling
app.use((err, req, res, next) => {
  console.error('Error:', err);
  res.status(err.status || 500).json({
    error: err.message || 'Internal server error',
    timestamp: new Date().toISOString()
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    error: 'Not found',
    path: req.path,
    timestamp: new Date().toISOString()
  });
});

const PORT = process.env.PORT || 3001;
const HOST = process.env.HOST || '0.0.0.0';

server.listen(PORT, HOST, () => {
  console.log(`Voice Ferry Web UI listening on http://${HOST}:${PORT}`);
  console.log(`WebSocket server ready for real-time updates`);
  
  // Initialize monitoring
  monitoringService.start();
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('Received SIGTERM, shutting down gracefully');
  monitoringService.stop();
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('Received SIGINT, shutting down gracefully');
  monitoringService.stop();
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});
