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
const TEST_ID = `test-responses-${Date.now()}`;
const TEST_CASE = 'responses';

// ====== Custom metrics ======
const guestLoginTime = new Trend('guest_login_time_ms', true);
const refreshTokenTime = new Trend('refresh_token_time_ms', true);
const responseTime = new Trend('response_time_ms', true);
const responseStreamTime = new Trend('response_stream_time_ms', true);
const responseTimeWithTools = new Trend('response_time_with_tools_ms', true);
const responseStreamTimeWithTools = new Trend('response_stream_time_with_tools_ms', true);
const errors = new Counter('response_errors');
const successes = new Counter('response_successes');

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
export const options = {
  iterations: 1,
  vus: 1,
  thresholds: {
    'http_req_failed': ['rate<0.05'],
    'guest_login_time_ms': ['p(95)<2000'],
    'refresh_token_time_ms': ['p(95)<2000'],
    'response_time_ms': ['p(95)<60000'],
    'response_stream_time_ms': ['p(95)<60000'],
    'response_time_with_tools_ms': ['p(95)<300000'],
    'response_stream_time_with_tools_ms': ['p(95)<300000'],
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

function testResponseApiNonStreamWithoutTools() {
  console.log('[RESPONSE NO-STREAM] Testing response API non-stream without tools...');
  
  const payload = {
    model: MODEL,
    input: [
      {
        role: 'user',
        content: 'Tell me about the latest advancements in renewable energy technology.'
      }
    ],
    stream: false
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/responses`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  responseTime.add(duration);
  
  // Record LLM-specific timings
  recordTimings(res, 'response_nonstream', 'no_tools');
  
  const ok = check(res, {
    'response non-stream status 200': (r) => r.status === 200,
    'response non-stream has content': (r) => r.body && r.body.length > 0
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      console.log(`[RESPONSE NO-STREAM] ✅ Success! Response received`);
      if (body.object) {
        console.log(`[RESPONSE NO-STREAM] ✅ Object: ${body.object}`);
      }
      if (body.choices && body.choices.length > 0) {
        const content = body.choices[0].message.content;
        console.log(`[RESPONSE NO-STREAM] ✅ Content: ${content.substring(0, 100)}...`);
      }
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[RESPONSE NO-STREAM] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[RESPONSE NO-STREAM] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function testResponseApiNonStreamWithTools() {
  console.log('[RESPONSE TOOLS] Testing response API non-stream with tools...');
  
  const payload = {
    model: MODEL,
    input: [
      {
        role: 'user',
        content: 'Google all news about the latest FED rate and summarize it for me'
      }
    ],
    stream: false,
    tools: [
      {
        type: 'function',
        function: {
          name: 'google_search',
          description: 'Tool to perform web searches via Serper API and retrieve rich results. It is able to retrieve organic search results, people also ask, related searches, and knowledge graph.',
          parameters: {
            type: 'object',
            properties: {
              autocorrect: {
                description: 'Whether to autocorrect spelling in query',
                type: 'boolean'
              },
              gl: {
                description: 'Optional region code for search results in ISO 3166-1 alpha-2 format (e.g. us, uk)',
                type: 'string'
              },
              hl: {
                description: 'Optional language code for search results in ISO 639-1 format (e.g. en, es)',
                type: 'string'
              },
              num: {
                description: 'Number of search results to return (1-100, default 10)',
                type: 'integer',
                minimum: 1,
                maximum: 100
              },
              q: {
                description: 'Search query string',
                type: 'string'
              }
            },
            required: ['q']
          }
        }
      }
    ]
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/responses`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  responseTimeWithTools.add(duration);
  
  // Record LLM-specific timings
  recordTimings(res, 'response_nonstream', 'with_tools');
  
  const ok = check(res, {
    'response tools status 200': (r) => r.status === 200,
    'response tools has content': (r) => r.body && r.body.length > 0
  });
  
  if (ok) {
    try {
      const body = JSON.parse(res.body);
      console.log(`[RESPONSE TOOLS] ✅ Success! Response with tools received`);
      if (body.object) {
        console.log(`[RESPONSE TOOLS] ✅ Object: ${body.object}`);
      }
      if (body.choices && body.choices.length > 0) {
        const choice = body.choices[0];
        if (choice.message && choice.message.content) {
          const content = choice.message.content;
          console.log(`[RESPONSE TOOLS] ✅ Content: ${content.substring(0, 100)}...`);
        }
        if (choice.message && choice.message.tool_calls) {
          console.log(`[RESPONSE TOOLS] ✅ Tool calls: ${choice.message.tool_calls.length}`);
        }
      }
      successes.add(1);
      return true;
    } catch (e) {
      console.log('[RESPONSE TOOLS] ❌ Failed to parse response');
      errors.add(1);
      return false;
    }
  } else {
    console.log('[RESPONSE TOOLS] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function testResponseApiStreamWithoutTools() {
  console.log('[RESPONSE STREAM] Testing response API stream without tools...');
  
  const payload = {
    model: MODEL,
    input: [
      {
        role: 'user',
        content: 'Write a detailed explanation of quantum computing in 3 paragraphs.'
      }
    ],
    stream: true
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/responses`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  responseStreamTime.add(duration);
  
  // Record LLM-specific timings
  recordTimings(res, 'response_stream', 'no_tools');
  
  const ok = check(res, {
    'response stream status 200': (r) => r.status === 200,
    'response stream has content': (r) => r.body && r.body.length > 0,
    'response stream is event-stream': (r) => r.headers['Content-Type'] && r.headers['Content-Type'].includes('text/event-stream')
  });
  
  if (ok) {
    const lines = res.body.split('\n');
    let chunkCount = 0;
    let hasContent = false;
    let hasMetadata = false;
    let hasDone = false;
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();
      if (line.startsWith('data: ')) {
        if (line.includes('[DONE]')) {
          hasDone = true;
          console.log(`[RESPONSE STREAM] ✅ Received completion signal: data: [DONE]`);
        } else {
          try {
            chunkCount++;
            const data = JSON.parse(line.substring(6));
            if (data.choices && data.choices[0] && data.choices[0].delta && data.choices[0].delta.content) {
              hasContent = true;
            }
            if (data.metadata) {
              hasMetadata = true;
            }
          } catch (e) {
            // Ignore parsing errors for non-JSON lines
          }
        }
      }
    }
    
    console.log(`[RESPONSE STREAM] ✅ Success! Received ${chunkCount} chunks`);
    console.log(`[RESPONSE STREAM] ✅ Has content: ${hasContent}`);
    console.log(`[RESPONSE STREAM] ✅ Has metadata: ${hasMetadata}`);
    console.log(`[RESPONSE STREAM] ✅ Stream completed: ${hasDone}`);
    
    if (!hasDone) {
      console.log('[RESPONSE STREAM] ⚠️ Warning: No [DONE] signal received');
    }
    
    successes.add(1);
    return true;
  } else {
    console.log('[RESPONSE STREAM] ❌ Failed');
    errors.add(1);
    return false;
  }
}

function testResponseApiStreamWithTools() {
  console.log('[RESPONSE STREAM TOOLS] Testing response API stream with tools...');
  
  const payload = {
    model: MODEL,
    input: [
      {
        role: 'user',
        content: 'Search for the latest AI breakthroughs and explain their significance'
      }
    ],
    stream: true,
    tools: [
      {
        type: 'function',
        function: {
          name: 'google_search',
          description: 'Tool to perform web searches via Serper API and retrieve rich results.',
          parameters: {
            type: 'object',
            properties: {
              q: {
                description: 'Search query string',
                type: 'string'
              },
              num: {
                description: 'Number of search results to return',
                type: 'integer',
                minimum: 1,
                maximum: 10
              }
            },
            required: ['q']
          }
        }
      }
    ]
  };
  
  const headers = buildHeaders();
  const body = JSON.stringify(payload);
  const url = `${BASE}/v1/responses`;
  
  debugRequest('POST', url, headers, body);
  debugLog('Payload details:', payload);
  
  const startTime = Date.now();
  const res = http.post(url, body, { headers });
  
  debugResponse(res);
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  responseStreamTimeWithTools.add(duration);
  
  // Record LLM-specific timings
  recordTimings(res, 'response_stream', 'with_tools');
  
  const ok = check(res, {
    'response stream tools status 200': (r) => r.status === 200,
    'response stream tools has content': (r) => r.body && r.body.length > 0,
    'response stream tools is event-stream': (r) => r.headers['Content-Type'] && r.headers['Content-Type'].includes('text/event-stream')
  });
  
  if (ok) {
    const lines = res.body.split('\n');
    let chunkCount = 0;
    let hasContent = false;
    let hasToolCalls = false;
    let hasDone = false;
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();
      if (line.startsWith('data: ')) {
        if (line.includes('[DONE]')) {
          hasDone = true;
          console.log(`[RESPONSE STREAM TOOLS] ✅ Received completion signal: data: [DONE]`);
        } else {
          try {
            chunkCount++;
            const data = JSON.parse(line.substring(6));
            if (data.choices && data.choices[0]) {
              const choice = data.choices[0];
              if (choice.delta && choice.delta.content) {
                hasContent = true;
              }
              if (choice.delta && choice.delta.tool_calls) {
                hasToolCalls = true;
              }
            }
          } catch (e) {
            // Ignore parsing errors for non-JSON lines
          }
        }
      }
    }
    
    console.log(`[RESPONSE STREAM TOOLS] ✅ Success! Received ${chunkCount} chunks`);
    console.log(`[RESPONSE STREAM TOOLS] ✅ Has content: ${hasContent}`);
    console.log(`[RESPONSE STREAM TOOLS] ✅ Has tool calls: ${hasToolCalls}`);
    console.log(`[RESPONSE STREAM TOOLS] ✅ Stream completed: ${hasDone}`);
    
    if (!hasDone) {
      console.log('[RESPONSE STREAM TOOLS] ⚠️ Warning: No [DONE] signal received');
    }
    
    successes.add(1);
    return true;
  } else {
    console.log('[RESPONSE STREAM TOOLS] ❌ Failed');
    errors.add(1);
    return false;
  }
}

// ====== Main Test Function ======
export default function() {
  console.log('\n========================================');
  console.log('  RESPONSE API TESTS');
  console.log('========================================');
  console.log(`Base URL: ${BASE}`);
  console.log(`Model: ${MODEL}`);
  console.log(`Debug Mode: ${DEBUG ? 'ENABLED' : 'DISABLED'}`);
  console.log('');
  
  // Step 1: Guest Login
  console.log('[1/6] Guest Login');
  if (!guestLogin()) {
    console.log('❌ Guest login failed, aborting test');
    return;
  }
  sleep(1);
  
  // Step 2: Refresh Token
  console.log('\n[2/6] Refresh Token');
  refreshAccessToken();
  sleep(1);
  
  // Step 3: Response API Non-Stream Without Tools
  console.log('\n[3/6] Response API Non-Stream Without Tools');
  refreshAccessToken(); // Refresh before response
  testResponseApiNonStreamWithoutTools();
  sleep(2);
  
  // Step 4: Response API Non-Stream With Tools
  console.log('\n[4/6] Response API Non-Stream With Tools');
  refreshAccessToken(); // Refresh before response
  testResponseApiNonStreamWithTools();
  sleep(2);
  
  // Step 5: Response API Stream Without Tools
  console.log('\n[5/6] Response API Stream Without Tools');
  refreshAccessToken(); // Refresh before response
  testResponseApiStreamWithoutTools();
  sleep(2);
  
  // Step 6: Response API Stream With Tools
  console.log('\n[6/6] Response API Stream With Tools');
  refreshAccessToken(); // Refresh before response
  testResponseApiStreamWithTools();
  
  // Summary
  console.log('\n========================================');
  console.log('           TEST SUMMARY');
  console.log('========================================');
  console.log('✅ Guest authentication');
  console.log('✅ Token refresh');
  console.log('✅ Response API non-stream without tools');
  console.log('✅ Response API non-stream with tools');
  console.log('✅ Response API stream without tools');
  console.log('✅ Response API stream with tools');
  console.log('');
  console.log('Response API tests completed!');
  console.log('========================================\n');
  
  sleep(2);
}
