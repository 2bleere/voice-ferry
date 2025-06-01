# Multi-stage Dockerfile for Voice Ferry Web UI

# Development stage
FROM node:18-alpine AS development

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies (including devDependencies)
RUN npm install

# Copy application code
COPY . .

# Create necessary directories
RUN mkdir -p /app/config /app/logs /app/data

# Expose ports (3000 for app, 9229 for debugger)
EXPOSE 3000 9229

# Start development server with nodemon
CMD ["npm", "run", "dev"]

# Production build stage
FROM node:18-alpine AS build

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install all dependencies for build
RUN npm ci

# Copy source code
COPY . .

# Run any build steps (if you add build scripts later)
# RUN npm run build

# Production stage
FROM node:18-alpine AS production

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install only production dependencies
RUN npm ci --only=production && npm cache clean --force

# Create non-root user for security
RUN addgroup -g 1001 -S nodejs && \
    adduser -S voice-ferry -u 1001 -G nodejs

# Copy application code from build stage
COPY --from=build --chown=voice-ferry:nodejs /app .

# Create necessary directories with proper permissions
RUN mkdir -p /app/config /app/logs /app/data && \
    chown -R voice-ferry:nodejs /app

# Switch to non-root user
USER voice-ferry

# Expose port
EXPOSE 3001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3001/api/health', (res) => { process.exit(res.statusCode === 200 ? 0 : 1) })"

# Start the application
CMD ["node", "server.js"]

# Default to production stage
FROM production
