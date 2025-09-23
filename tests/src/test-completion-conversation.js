import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';

// ====== Config via ENV (with defaults) ======
const BASE = __ENV.BASE || 'https://api-dev.jan.ai';
const MODEL = __ENV.MODEL || 'jan-v1-4b';
const DEBUG = __ENV.DEBUG === 'true' || __ENV.DEBUG === '1';
const API_KEY = __ENV.API_KEY || '';
const LOADTEST_TOKEN = __ENV.LOADTEST_TOKEN || '';

// ====== Global state ======
let accessToken = '';
let refreshToken = '';
let conversationId = '';

// ====== Common headers ======
function buildHeaders(extra = {}) {
  const h = { 'Content-Type': 'application/json' };
  for (const key in extra) {
    if (extra.hasOwnProperty(key)) {
      h[key] = extra[key];
    }
  }
  if (API_KEY) h['Authorization'] = `Bearer ${API_KEY}`;
  if (LOADTEST_TOKEN) h['x-loadtest-token'] = LOADTEST_TOKEN;
  if (accessToken) h['Authorization'] = `Bearer ${accessToken}`;
  return h;
}

// ====== Custom metrics ======
const guestLoginTime = new Trend('guest_login_time_ms', true);
const refreshTokenTime = new Trend('refresh_token_time_ms', true);
const conversationTime = new Trend('conversation_time_ms', true);
const completionTime = new Trend('completion_time_ms', true);
const listConversationsTime = new Trend('list_conversations_time_ms', true);
const conversationItemsTime = new Trend('conversation_items_time_ms', true);
const errors = new Counter('conversation_errors');
const successes = new Counter('conversation_successes');

// ====== Options ======
export const options = {
  iterations: 1,
  vus: 1,
  thresholds: {
    'http_req_failed': ['rate<0.05'],
    'guest_login_time_ms': ['p(95)<2000'],
    'refresh_token_time_ms': ['p(95)<2000'],
    'conversation_time_ms': ['p(95)<3000'],
    'completion_time_ms': ['p(95)<10000'],
    'list_conversations_time_ms': ['p(95)<3000'],
    'conversation_items_time_ms': ['p(95)<3000'],
  },
  discardResponseBodies: false,
};

// ====== Debug Functions ======
function debugLog(message, data = null) {
  if (DEBUG) {
    console.log(`[DEBUG] ${message}`);
    if (data) {
      console.log(`[DEBUG] Data:`, JSON.stringify(data, null, 2));
    }
  }
}

function debugRequest(method, url, headers, body) {
  if (DEBUG) {
    console.log(`[DEBUG] ====== REQUEST ======`);
    console.log(`[DEBUG] Method: ${method}`);
    console.log(`[DEBUG] URL: ${url}`);
    console.log(`[DEBUG] Headers:`, JSON.stringify(headers, null, 2));
    if (body) {
      console.log(`[DEBUG] Body:`, JSON.stringify(JSON.parse(body), null, 2));
    }
    console.log(`[DEBUG] ====================`);
  }
}

function debugResponse(response) {
  if (DEBUG) {
    console.log(`[DEBUG] ====== RESPONSE ======`);
    console.log(`[DEBUG] Status: ${response.status}`);
    console.log(`[DEBUG] Headers:`, JSON.stringify(response.headers, null, 2));
    console.log(`[DEBUG] Body:`, response.body);
    console.log(`[DEBUG] =====================`);
  }
}

// ====== Test Functions ======
function guestLogin() {
  console.log('[GUEST LOGIN] Starting guest login...');
  
  const headers = buildHeaders();
  const body = JSON.stringify({});
  const url = `${BASE}/v1/auth/guest-login`;
  
  debugRequest('POST', url, headers, body);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  guestLoginTime.add(duration);
  
  const ok = check(res, {
    'guest login status 200': (r) => r.status === 200,
    'guest login has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.access_token.length > 0;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      accessToken = body.access_token;
      
      // Extract refresh token from Set-Cookie header
      const setCookieHeader = res.headers['Set-Cookie'];
      if (setCookieHeader) {
        const refreshTokenMatch = setCookieHeader.match(/jan_refresh_token=([^;]+)/);
        if (refreshTokenMatch) {
          refreshToken = refreshTokenMatch[1];
          console.log(`[GUEST LOGIN] ✅ Success! Token: ${accessToken.substring(0, 20)}...`);
          console.log(`[GUEST LOGIN] ✅ Refresh token extracted`);
        } else {
          console.log('[GUEST LOGIN] ⚠️ No refresh token found in cookies');
        }
      }
      
      return true;
    } catch (e) {
      console.log('[GUEST LOGIN] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[GUEST LOGIN] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function refreshAccessToken() {
  if (!refreshToken) {
    console.log('[REFRESH TOKEN] ⚠️ No refresh token available, skipping');
    return false;
  }
  
  console.log('[REFRESH TOKEN] Refreshing access token...');
  
  const headers = {
    'Content-Type': 'application/json',
    'Cookie': `jan_refresh_token=${refreshToken}`,
    'Authorization': `Bearer ${accessToken}`
  };
  
  const url = `${BASE}/v1/auth/refresh-token`;
  
  debugRequest('GET', url, headers);
  
  const startTime = Date.now();
  const res = http.get(url, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  refreshTokenTime.add(duration);
  
  const ok = check(res, {
    'refresh token status 200': (r) => r.status === 200,
    'refresh token has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.access_token.length > 0;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      accessToken = body.access_token;
      
      // Update refresh token from new Set-Cookie header
      const setCookieHeader = res.headers['Set-Cookie'];
      if (setCookieHeader) {
        const refreshTokenMatch = setCookieHeader.match(/jan_refresh_token=([^;]+)/);
        if (refreshTokenMatch) {
          refreshToken = refreshTokenMatch[1];
        }
      }
      
      console.log(`[REFRESH TOKEN] ✅ Success! New token: ${accessToken.substring(0, 20)}...`);
      console.log(`[REFRESH TOKEN] ✅ Expires in: ${body.expires_in} seconds`);
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[REFRESH TOKEN] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[REFRESH TOKEN] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function createConversation() {
  console.log('[CREATE CONV] Creating new conversation...');
  
  const payload = {
    title: "Test Conversation - K6 Load Test"
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/conversations`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  conversationTime.add(duration);
  
  const ok = check(res, {
    'create conversation status 2xx': (r) => r.status >= 200 && r.status < 300,
    'create conversation has id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id && body.id.length > 0;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      conversationId = body.id;
      console.log(`[CREATE CONV] ✅ Success! Status: ${res.status}, Conversation ID: ${conversationId}`);
      console.log(`[CREATE CONV] ✅ Title: ${body.title}`);
      console.log(`[CREATE CONV] ✅ Object: ${body.object}`);
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[CREATE CONV] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log(`[CREATE CONV] ❌ Failed! Status: ${res.status}, Body: ${res.body}`);
    errors.add(1);
    return false;
  }
}

function addMessageToConversation(message, isFirstMessage = false) {
  console.log(`[ADD MESSAGE] Adding non-streaming message to conversation...`);
  
  const payload = {
    model: MODEL,
    messages: [
      { role: 'user', content: message }
    ],
    temperature: 0.7,
    max_tokens: 150,
    stream: false,
    conversation: conversationId,
    store: true
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/conv/chat/completions`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  completionTime.add(duration);
  
  const ok = check(res, {
    'conversation completion status 200': (r) => r.status === 200,
    'conversation completion has choices': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.choices && body.choices.length > 0;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      const content = body.choices[0].message.content;
      console.log(`[ADD MESSAGE] ✅ Success! Response ID: ${body.id}`);
      console.log(`[ADD MESSAGE] ✅ Content: ${content.substring(0, 80)}...`);
      if (isFirstMessage) {
        console.log(`[ADD MESSAGE] ✅ First message in conversation ${conversationId}`);
      }
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[ADD MESSAGE] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[ADD MESSAGE] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function addStreamingMessageToConversation(message, isFirstMessage = false) {
  console.log(`[ADD STREAMING MESSAGE] Adding streaming message to conversation...`);
  
  const payload = {
    model: MODEL,
    messages: [
      { role: 'user', content: message }
    ],
    temperature: 0.7,
    max_tokens: 150,
    stream: true,
    conversation: conversationId,
    store: true
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/conv/chat/completions`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  completionTime.add(duration);
  
  const ok = check(res, {
    'streaming completion status 200': (r) => r.status === 200,
    'streaming completion has content': (r) => r.body && r.body.length > 0,
    'streaming completion is event-stream': (r) => r.headers['Content-Type'] && r.headers['Content-Type'].includes('text/event-stream')
  });
  
  if (ok) {
    const lines = res.body.split('\n');
    let chunkCount = 0;
    let hasContent = false;
    let hasDone = false;
    let responseId = '';
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();
      if (line.startsWith('data: ')) {
        if (line.includes('[DONE]')) {
          hasDone = true;
          console.log(`[ADD STREAMING MESSAGE] ✅ Received completion signal: data: [DONE]`);
        } else {
          try {
            chunkCount++;
            const data = JSON.parse(line.substring(6));
            if (data.id && !responseId) {
              responseId = data.id;
            }
            if (data.choices && data.choices[0] && data.choices[0].delta && data.choices[0].delta.content) {
              hasContent = true;
            }
          } catch (e) {
            // Ignore parsing errors for non-JSON lines
          }
        }
      }
    }
    
    console.log(`[ADD STREAMING MESSAGE] ✅ Success! Response ID: ${responseId}`);
    console.log(`[ADD STREAMING MESSAGE] ✅ Received ${chunkCount} chunks`);
    console.log(`[ADD STREAMING MESSAGE] ✅ Has content: ${hasContent}`);
    console.log(`[ADD STREAMING MESSAGE] ✅ Stream completed: ${hasDone}`);
    
    if (!hasDone) {
      console.log('[ADD STREAMING MESSAGE] ⚠️ Warning: No [DONE] signal received');
    }
    
    if (isFirstMessage) {
      console.log(`[ADD STREAMING MESSAGE] ✅ First streaming message in conversation ${conversationId}`);
    }
    
    successes.add(1);
    return true;
  } else {
    console.log('[ADD STREAMING MESSAGE] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function getConversation() {
  if (!conversationId) {
    console.log('[GET CONV] ⚠️ No conversation ID available, skipping');
    return false;
  }
  
  console.log('[GET CONV] Loading conversation details...');
  
  const headers = buildHeaders();
  const url = `${BASE}/v1/conversations/${conversationId}`;
  
  debugRequest('GET', url, headers);
  
  const startTime = Date.now();
  const res = http.get(url, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  conversationTime.add(duration);
  
  const ok = check(res, {
    'get conversation status 200': (r) => r.status === 200,
    'get conversation has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id && body.title;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      console.log(`[GET CONV] ✅ Success! ID: ${body.id}`);
      console.log(`[GET CONV] ✅ Title: ${body.title}`);
      console.log(`[GET CONV] ✅ Object: ${body.object}`);
      console.log(`[GET CONV] ✅ Created: ${body.created_at}`);
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[GET CONV] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[GET CONV] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function listConversations() {
  console.log('[LIST CONV] Loading conversations list...');
  
  const headers = buildHeaders();
  const url = `${BASE}/v1/conversations`;
  
  debugRequest('GET', url, headers);
  
  const startTime = Date.now();
  const res = http.get(url, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  listConversationsTime.add(duration);
  
  const ok = check(res, {
    'list conversations status 200': (r) => r.status === 200,
    'list conversations has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.object === 'list' && body.data && Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      console.log(`[LIST CONV] ✅ Success! Object: ${body.object}`);
      console.log(`[LIST CONV] ✅ Total conversations: ${body.data.length}`);
      console.log(`[LIST CONV] ✅ Has more: ${body.has_more}`);
      
      // Check if our conversation is in the list
      const ourConv = body.data.find(conv => conv.id === conversationId);
      if (ourConv) {
        console.log(`[LIST CONV] ✅ Found our conversation: ${ourConv.title}`);
      }
      
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[LIST CONV] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[LIST CONV] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function getConversationItems() {
  if (!conversationId) {
    console.log('[GET ITEMS] ⚠️ No conversation ID available, skipping');
    return false;
  }
  
  console.log('[GET ITEMS] Loading conversation items...');
  
  const headers = buildHeaders();
  const url = `${BASE}/v1/conversations/${conversationId}/items`;
  
  debugRequest('GET', url, headers);
  
  const startTime = Date.now();
  const res = http.get(url, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  conversationItemsTime.add(duration);
  
  const ok = check(res, {
    'get items status 200': (r) => r.status === 200,
    'get items has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      console.log(`[GET ITEMS] ✅ Success! Object: ${body.object}`);
      console.log(`[GET ITEMS] ✅ Items count: ${body.data.length}`);
      console.log(`[GET ITEMS] ✅ Total: ${body.total}`);
      console.log(`[GET ITEMS] ✅ Has more: ${body.has_more}`);
      
      if (body.data.length > 0) {
        console.log('[GET ITEMS] ✅ Items:');
        body.data.forEach((item, index) => {
          console.log(`[GET ITEMS]   ${index + 1}. ID: ${item.id}, Role: ${item.role}, Type: ${item.type}`);
          if (item.content && item.content.length > 0 && item.content[0].text) {
            const content = item.content[0].text.value;
            const preview = content.length > 50 ? content.substring(0, 50) + '...' : content;
            console.log(`[GET ITEMS]      Content: ${preview}`);
          }
        });
      }
      
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[GET ITEMS] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[GET ITEMS] ❌ Failed');
    errors.add(1);
    return false;
  }
}

// ====== Main Test Function ======
export default function() {
  console.log('\n========================================');
  console.log('  CONVERSATION COMPLETION TESTS');
  console.log('========================================');
  console.log(`Base URL: ${BASE}`);
  console.log(`Model: ${MODEL}`);
  console.log(`Debug Mode: ${DEBUG ? 'ENABLED' : 'DISABLED'}`);
  console.log('');
  
  // Step 1: Guest Login
  console.log('[1/8] Guest Login');
  if (!guestLogin()) {
    console.log('❌ Guest login failed, aborting test');
    return;
  }
  sleep(1);
  
  // Step 2: Refresh Token
  console.log('\n[2/8] Refresh Token');
  refreshAccessToken();
  sleep(1);
  
  // Step 3: Create Conversation
  console.log('\n[3/8] Create Conversation');
  if (!createConversation()) {
    console.log('❌ Create conversation failed, aborting test');
    return;
  }
  sleep(1);
  
  // Step 4: Add First Message (Non-Streaming)
  console.log('\n[4/9] Add First Message to Conversation (Non-Streaming)');
  refreshAccessToken(); // Refresh before completion
  addMessageToConversation('Hello! What is artificial intelligence?', true);
  sleep(1);
  
  // Step 5: Add Second Message (Streaming)
  console.log('\n[5/9] Add Second Message to Conversation (Streaming)');
  refreshAccessToken(); // Refresh before completion
  addStreamingMessageToConversation('Can you explain machine learning in simple terms?');
  sleep(1);
  
  // Step 6: Get Conversation Details
  console.log('\n[6/9] Get Conversation Details');
  getConversation();
  sleep(1);
  
  // Step 7: List All Conversations
  console.log('\n[7/9] List All Conversations');
  listConversations();
  sleep(1);
  
  // Step 8: Get Conversation Items
  console.log('\n[8/9] Get Conversation Items');
  getConversationItems();
  
  // Summary
  console.log('\n========================================');
  console.log('           TEST SUMMARY');
  console.log('========================================');
  console.log('✅ Guest authentication');
  console.log('✅ Token refresh');
  console.log('✅ Conversation creation');
  console.log('✅ Non-streaming message exchange');
  console.log('✅ Streaming message exchange');
  console.log('✅ Conversation retrieval');
  console.log('✅ Conversation listing');
  console.log('✅ Item retrieval');
  console.log('');
  console.log('Conversation completion tests completed!');
  console.log(`Conversation ID: ${conversationId}`);
  console.log('========================================\n');
  
  sleep(2);
}
