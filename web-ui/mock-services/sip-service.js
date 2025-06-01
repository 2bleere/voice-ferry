/**
 * Mock SIP Service for Development and Testing
 * 
 * This service simulates a minimal SIP authentication and registration service
 * for testing the SIP Users functionality of the web UI.
 */

const dgram = require('dgram');
const server = dgram.createSocket('udp4');
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const fs = require('fs');
const path = require('path');

// In-memory storage for SIP users
let sipUsers = [
  {
    id: '1',
    username: 'user1',
    password: 'test123',
    domain: 'example.com',
    enabled: true,
    realm: 'voice-ferry',
    maxSessions: 5,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  },
  {
    id: '2',
    username: 'user2',
    password: 'test456',
    domain: 'example.com',
    enabled: true,
    realm: 'voice-ferry',
    maxSessions: 3,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  },
  {
    id: '3',
    username: 'disabled',
    password: 'test789',
    domain: 'example.com',
    enabled: false,
    realm: 'voice-ferry',
    maxSessions: 1,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  }
];

// Start the UDP server for SIP
server.on('error', (err) => {
  console.error(`SIP server error: ${err.stack}`);
  server.close();
});

server.on('message', (msg, rinfo) => {
  console.log(`SIP message received from ${rinfo.address}:${rinfo.port}`);
  console.log(`Message: ${msg.toString()}`);
  
  // Simple echo response for now
  const response = `SIP/2.0 200 OK
Via: SIP/2.0/UDP ${rinfo.address}:${rinfo.port}
From: Mock SIP Server <sip:server@voice-ferry.local>
To: <sip:client@${rinfo.address}>
Call-ID: mock-call-id
CSeq: 1 REGISTER
Contact: <sip:server@voice-ferry.local>
Content-Length: 0

`;
  
  server.send(response, rinfo.port, rinfo.address, (err) => {
    if (err) {
      console.error(`Error sending SIP response: ${err}`);
    } else {
      console.log(`SIP response sent to ${rinfo.address}:${rinfo.port}`);
    }
  });
});

server.on('listening', () => {
  const address = server.address();
  console.log(`SIP server listening on ${address.address}:${address.port}`);
});

server.bind(5060);

// Setup gRPC server for API calls
const PROTO_PATH = path.resolve(__dirname, '../proto/b2bua.proto');
let packageDefinition;

try {
  // Try to load the proto file
  if (fs.existsSync(PROTO_PATH)) {
    packageDefinition = protoLoader.loadSync(PROTO_PATH, {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true
    });
  } else {
    // Create a minimal proto definition for testing
    const mockProtoContent = `
    syntax = "proto3";
    package b2bua;
    
    service SipUserService {
      rpc GetSipUsers(GetSipUsersRequest) returns (GetSipUsersResponse) {}
      rpc CreateSipUser(CreateSipUserRequest) returns (CreateSipUserResponse) {}
      rpc UpdateSipUser(UpdateSipUserRequest) returns (UpdateSipUserResponse) {}
      rpc DeleteSipUser(DeleteSipUserRequest) returns (DeleteSipUserResponse) {}
    }
    
    message SipUser {
      string id = 1;
      string username = 2;
      string password = 3;
      string domain = 4;
      bool enabled = 5;
      string realm = 6;
      int32 max_sessions = 7;
      string created_at = 8;
      string updated_at = 9;
    }
    
    message GetSipUsersRequest {}
    
    message GetSipUsersResponse {
      repeated SipUser users = 1;
      bool success = 2;
      string error = 3;
    }
    
    message CreateSipUserRequest {
      SipUser user = 1;
    }
    
    message CreateSipUserResponse {
      SipUser user = 1;
      bool success = 2;
      string error = 3;
    }
    
    message UpdateSipUserRequest {
      SipUser user = 1;
    }
    
    message UpdateSipUserResponse {
      SipUser user = 1;
      bool success = 2;
      string error = 3;
    }
    
    message DeleteSipUserRequest {
      string id = 1;
    }
    
    message DeleteSipUserResponse {
      bool success = 1;
      string error = 2;
    }
    `;
    
    // Write mock proto file
    const mockProtoPath = path.resolve(__dirname, 'mock-b2bua.proto');
    fs.writeFileSync(mockProtoPath, mockProtoContent);
    
    packageDefinition = protoLoader.loadSync(mockProtoPath, {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true
    });
    
    console.log('Created mock proto definition for testing');
  }
  
  const grpcObject = grpc.loadPackageDefinition(packageDefinition);
  const b2buaProto = grpcObject.b2bua;
  
  const server = new grpc.Server();
  
  // Implement the service
  server.addService(b2buaProto.SipUserService.service, {
    GetSipUsers: (call, callback) => {
      console.log('gRPC GetSipUsers called');
      callback(null, { users: sipUsers, success: true });
    },
    
    CreateSipUser: (call, callback) => {
      console.log('gRPC CreateSipUser called');
      const user = call.request.user;
      user.id = (sipUsers.length + 1).toString();
      user.created_at = new Date().toISOString();
      user.updated_at = new Date().toISOString();
      
      sipUsers.push(user);
      callback(null, { user, success: true });
    },
    
    UpdateSipUser: (call, callback) => {
      console.log('gRPC UpdateSipUser called');
      const updatedUser = call.request.user;
      const index = sipUsers.findIndex(u => u.id === updatedUser.id);
      
      if (index !== -1) {
        updatedUser.updated_at = new Date().toISOString();
        sipUsers[index] = updatedUser;
        callback(null, { user: updatedUser, success: true });
      } else {
        callback(null, { success: false, error: 'User not found' });
      }
    },
    
    DeleteSipUser: (call, callback) => {
      console.log('gRPC DeleteSipUser called');
      const id = call.request.id;
      const initialLength = sipUsers.length;
      
      sipUsers = sipUsers.filter(user => user.id !== id);
      
      if (sipUsers.length < initialLength) {
        callback(null, { success: true });
      } else {
        callback(null, { success: false, error: 'User not found' });
      }
    }
  });
  
  // Start gRPC server
  server.bindAsync('0.0.0.0:50051', grpc.ServerCredentials.createInsecure(), (err, port) => {
    if (err) {
      console.error(`Failed to bind gRPC server: ${err}`);
      return;
    }
    
    server.start();
    console.log(`gRPC server running on port ${port}`);
  });
} catch (error) {
  console.error(`Error setting up gRPC server: ${error.message}`);
  console.error('Running in minimal mode with only SIP functionality');
}
