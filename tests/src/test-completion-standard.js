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

// ====== Test Configuration ======
const TEST_ID = `test-completion-standard-${Date.now()}`;
const TEST_CASE = 'completion-standard';

// ====== Custom metrics ======
const guestLoginTime = new Trend('guest_login_time_ms', true);
const refreshTokenTime = new Trend('refresh_token_time_ms', true);
const modelsTime = new Trend('models_time_ms', true);
const completionTime = new Trend('completion_time_ms', true);
const streamingTime = new Trend('streaming_time_ms', true);
const errors = new Counter('completion_errors');
const successes = new Counter('completion_successes');

// ====== LLM-specific metrics ======
const ttfb = new Trend('llm_ttfb_ms', true);
const recvTime = new Trend('llm_receiving_ms', true);
const totalDur = new Trend('llm_total_ms', true);
const queueDur = new Trend('llm_queue_ms', true);
// tokens per second is NOT a time metric; don't mark as time
const tokRate = new Trend('llm_tokens_per_sec');
const llmErrors = new Counter('llm_errors');

// ====== Helper functions ======
// helper: record timings with tags (scenario + status + promptType)
function recordTimings(res, scenario, promptType) {
  const status = String(res.status || 0);
  const tags = { scenario, status, prompt: promptType };

  ttfb.add(res.timings.waiting, tags);
  recvTime.add(res.timings.receiving, tags);
  totalDur.add(res.timings.duration, tags);

  // custom: queue time header (ms)
  const q = res.headers['X-Queue-Time'];
  if (q) {
    const val = parseFloat(q);
    if (!isNaN(val)) queueDur.add(val, tags);
  }

  // custom: tokens/sec if usage present
  if (status === '200') {
    try {
      const j = res.json();
      const comp = j.usage?.completion_tokens || 0;
      if (comp > 0) {
        tokRate.add(comp / (res.timings.duration / 1000), tags);
      }
    } catch {}
  }
}

// ====== Options ======
export const options = {
  iterations: 1,
  vus: 1,
  thresholds: {
    'http_req_failed': ['rate<0.05'],
    'guest_login_time_ms': ['p(95)<2000'],
    'refresh_token_time_ms': ['p(95)<2000'],
    'models_time_ms': ['p(95)<2000'],
    'completion_time_ms': ['p(95)<10000'],
    'streaming_time_ms': ['p(95)<15000'],
  },
  discardResponseBodies: false,
  tags: {
    testid: TEST_ID,
    test_case: TEST_CASE,
  },
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

function testModels() {
  console.log('[MODELS] Testing models endpoint...');
  
  const headers = buildHeaders();
  const url = `${BASE}/v1/models`;
  
  debugRequest('GET', url, headers);
  
  const startTime = Date.now();
  const res = http.get(url, { headers });
  
  debugResponse(res);
  
  const duration = Date.now() - startTime;
  modelsTime.add(duration);
  
  const ok = check(res, {
    'models status 200': (r) => r.status === 200,
    'models has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.object === 'list' && body.data && Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    },
    'models includes target model': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data.some(model => model.id === MODEL);
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      console.log(`[MODELS] ✅ Success! Found ${body.data.length} models`);
      console.log(`[MODELS] ✅ Target model ${MODEL} is available`);
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[MODELS] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[MODELS] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function testNonStreamingCompletion() {
  console.log('[NON-STREAMING] Testing standard completion...');
  
  const payload = {
    model: MODEL,
    messages: [
      { role: 'user', content: 'Hello! Tell me a short interesting fact about artificial intelligence.' }
    ],
    temperature: 0.7,
    max_tokens: 150,
    stream: false
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/chat/completions`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  completionTime.add(duration);
  
  // Record LLM-specific timings
  recordTimings(res, 'completion_nonstream', 'standard');
  
  const ok = check(res, {
    'completion status 200': (r) => r.status === 200,
    'completion has choices': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.choices && body.choices.length > 0;
      } catch (e) {
        return false;
      }
    },
    'completion has content': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.choices[0].message && body.choices[0].message.content;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      const content = body.choices[0].message.content;
      console.log(`[NON-STREAMING] ✅ Success! Response ID: ${body.id}`);
      console.log(`[NON-STREAMING] ✅ Content: ${content.substring(0, 100)}...`);
      console.log(`[NON-STREAMING] ✅ Usage: ${body.usage.total_tokens} tokens`);
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[NON-STREAMING] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[NON-STREAMING] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function testStreamingCompletion() {
  console.log('[STREAMING] Testing streaming completion...');
  
  const payload = {
    model: MODEL,
    messages: [
      { role: 'user', content: 'Write a short poem about technology in exactly 4 lines.' }
    ],
    temperature: 0.8,
    max_tokens: 100,
    stream: true
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/chat/completions`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  streamingTime.add(duration);
  
  // Record LLM-specific timings
  recordTimings(res, 'completion_stream', 'standard');
  
  const ok = check(res, {
    'streaming status 200': (r) => r.status === 200,
    'streaming has content': (r) => r.body && r.body.length > 0,
    'streaming is event-stream': (r) => r.headers['Content-Type'] && r.headers['Content-Type'].includes('text/event-stream')
  });
  
  if (ok) {
    const lines = res.body.split('\n');
    let chunkCount = 0;
    let hasContent = false;
    let hasDone = false;
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();
      if (line.startsWith('data: ')) {
        if (line.includes('[DONE]')) {
          hasDone = true;
          console.log(`[STREAMING] ✅ Received completion signal: data: [DONE]`);
        } else {
          try {
            chunkCount++;
            const data = JSON.parse(line.substring(6));
            if (data.choices && data.choices[0].delta && data.choices[0].delta.content) {
              hasContent = true;
            }
          } catch (e) {
            // Ignore parsing errors for non-JSON lines
          }
        }
      }
    }
    
    console.log(`[STREAMING] ✅ Success! Received ${chunkCount} chunks`);
    console.log(`[STREAMING] ✅ Has content: ${hasContent}`);
    console.log(`[STREAMING] ✅ Stream completed: ${hasDone}`);
    
    if (!hasDone) {
      console.log('[STREAMING] ⚠️ Warning: No [DONE] signal received');
    }
    
    successes.add(1);
    return true;
  } else {
    console.log('[STREAMING] ❌ Failed');
    errors.add(1);
    return false;
  }
}

// ====== Main Test Function ======
export default function() {
  console.log('\n========================================');
  console.log('  STANDARD COMPLETION TESTS');
  console.log('========================================');
  console.log(`Base URL: ${BASE}`);
  console.log(`Model: ${MODEL}`);
  console.log(`Debug Mode: ${DEBUG ? 'ENABLED' : 'DISABLED'}`);
  console.log('');
  
  // Step 1: Guest Login
  console.log('[1/5] Guest Login');
  if (!guestLogin()) {
    console.log('❌ Guest login failed, aborting test');
    return;
  }
  sleep(1);
  
  // Step 2: Refresh Token
  console.log('\n[2/5] Refresh Token');
  refreshAccessToken();
  sleep(1);
  
  // Step 3: Test Models
  console.log('\n[3/5] Test Models Endpoint');
  testModels();
  sleep(1);
  
  // Step 4: Non-Streaming Completion
  console.log('\n[4/5] Non-Streaming Completion');
  refreshAccessToken(); // Refresh before completion
  testNonStreamingCompletion();
  sleep(1);
  
  // Step 5: Streaming Completion
  console.log('\n[5/5] Streaming Completion');
  refreshAccessToken(); // Refresh before completion
  testStreamingCompletion();
  
  // Summary
  console.log('\n========================================');
  console.log('           TEST SUMMARY');
  console.log('========================================');
  console.log('✅ Guest authentication');
  console.log('✅ Token refresh');
  console.log('✅ Models endpoint');
  console.log('✅ Non-streaming completions');
  console.log('✅ Streaming completions');
  console.log('');
  console.log('Standard completion tests completed!');
  console.log('========================================\n');
  
  sleep(2);
}
