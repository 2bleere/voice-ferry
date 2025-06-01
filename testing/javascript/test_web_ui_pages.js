#!/usr/bin/env node

/**
 * Test script to verify the web UI pages implementation
 */

const http = require('http');
const fs = require('fs');
const path = require('path');

const BASE_URL = 'http://localhost:3001';

async function makeRequest(url, options = {}) {
  return new Promise((resolve, reject) => {
    const req = http.request(url, options, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          data: data
        });
      });
    });
    
    req.on('error', reject);
    
    if (options.body) {
      req.write(options.body);
    }
    
    req.end();
  });
}

async function testLogin() {
  console.log('🔐 Testing login...');
  
  try {
    const response = await makeRequest(`${BASE_URL}/api/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        username: 'admin',
        password: 'admin123'
      })
    });
    
    console.log(`Status: ${response.statusCode}`);
    
    if (response.statusCode === 200) {
      const data = JSON.parse(response.data);
      if (data.success && data.token) {
        console.log('✅ Login successful');
        return data.token;
      }
    }
    
    console.log('❌ Login failed');
    console.log('Response:', response.data);
    return null;
  } catch (error) {
    console.log('❌ Login error:', error.message);
    return null;
  }
}

async function testAPI(endpoint, token) {
  console.log(`📡 Testing ${endpoint}...`);
  
  try {
    const response = await makeRequest(`${BASE_URL}${endpoint}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    console.log(`Status: ${response.statusCode}`);
    
    if (response.statusCode === 200) {
      console.log('✅ API endpoint working');
      return true;
    } else {
      console.log('❌ API endpoint failed');
      console.log('Response:', response.data);
      return false;
    }
  } catch (error) {
    console.log('❌ API error:', error.message);
    return false;
  }
}

async function testHealthCheck() {
  console.log('🏥 Testing health check...');
  
  try {
    const response = await makeRequest(`${BASE_URL}/health`);
    console.log(`Status: ${response.statusCode}`);
    
    if (response.statusCode === 200) {
      console.log('✅ Health check passed');
      return true;
    } else {
      console.log('❌ Health check failed');
      return false;
    }
  } catch (error) {
    console.log('❌ Health check error:', error.message);
    return false;
  }
}

async function testStaticFiles() {
  console.log('📁 Testing static files...');
  
  const filesToCheck = [
    '/js/metrics.js',
    '/js/logs.js', 
    '/js/alerts.js',
    '/index.html'
  ];
  
  let allGood = true;
  
  for (const file of filesToCheck) {
    try {
      const response = await makeRequest(`${BASE_URL}${file}`);
      if (response.statusCode === 200) {
        console.log(`✅ ${file} accessible`);
      } else {
        console.log(`❌ ${file} not accessible (${response.statusCode})`);
        allGood = false;
      }
    } catch (error) {
      console.log(`❌ ${file} error:`, error.message);
      allGood = false;
    }
  }
  
  return allGood;
}

async function verifyImplementation() {
  console.log('📋 Verifying implementation files...');
  
  const webUIDir = '/Users/wiredboy/Documents/git_live/go-voice-ferry copy/web-ui/public';
  
  const filesToCheck = [
    path.join(webUIDir, 'js/metrics.js'),
    path.join(webUIDir, 'js/logs.js'),
    path.join(webUIDir, 'js/alerts.js'),
    path.join(webUIDir, 'index.html')
  ];
  
  let allExist = true;
  
  for (const file of filesToCheck) {
    try {
      const stats = fs.statSync(file);
      const size = Math.round(stats.size / 1024);
      console.log(`✅ ${path.basename(file)} exists (${size}KB)`);
    } catch (error) {
      console.log(`❌ ${path.basename(file)} missing`);
      allExist = false;
    }
  }
  
  return allExist;
}

async function main() {
  console.log('🚀 Starting Web UI Pages Test');
  console.log('================================');
  
  // First verify files exist
  const filesExist = await verifyImplementation();
  if (!filesExist) {
    console.log('❌ Some implementation files are missing');
    process.exit(1);
  }
  
  // Test health check
  const healthOk = await testHealthCheck();
  if (!healthOk) {
    console.log('❌ Web UI service is not responding');
    process.exit(1);
  }
  
  // Test login
  const token = await testLogin();
  if (!token) {
    console.log('❌ Authentication failed');
    process.exit(1);
  }
  
  // Test API endpoints
  const endpoints = [
    '/api/metrics',
    '/api/logs',
    '/api/alerts'
  ];
  
  let apiSuccess = true;
  for (const endpoint of endpoints) {
    const success = await testAPI(endpoint, token);
    if (!success) apiSuccess = false;
  }
  
  // Test static files
  const staticOk = await testStaticFiles();
  
  console.log('\n📊 Test Results:');
  console.log('================');
  console.log(`Files exist: ${filesExist ? '✅' : '❌'}`);
  console.log(`Health check: ${healthOk ? '✅' : '❌'}`);
  console.log(`Authentication: ${token ? '✅' : '❌'}`);
  console.log(`API endpoints: ${apiSuccess ? '✅' : '❌'}`);
  console.log(`Static files: ${staticOk ? '✅' : '❌'}`);
  
  if (filesExist && healthOk && token && apiSuccess && staticOk) {
    console.log('\n🎉 All tests passed! The implementation appears to be working correctly.');
  } else {
    console.log('\n⚠️  Some tests failed. Check the output above for details.');
  }
}

main().catch(console.error);
