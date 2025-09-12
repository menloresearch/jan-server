import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';

// ====== Config via ENV (with defaults) ======
const BASE = __ENV.BASE || 'https://api-dev.jan.ai';
const MODEL = __ENV.MODEL || 'jan-v1-4b';
const TARGET_NONSTREAM_RPS = Number(__ENV.NONSTREAM_RPS || 2);
const TARGET_STREAM_RPS    = Number(__ENV.STREAM_RPS    || 1);
const DURATION_MINUTES     = Number(__ENV.DURATION_MIN || 5);
const LOADTEST_TOKEN       = __ENV.LOADTEST_TOKEN || '';
const API_KEY              = __ENV.API_KEY || '';

// ====== Common headers ======
function buildHeaders(extra = {}) {
  const h = { 'Content-Type': 'application/json', ...extra };
  if (API_KEY) h['Authorization'] = `Bearer ${API_KEY}`;
  if (LOADTEST_TOKEN) h['x-loadtest-token'] = LOADTEST_TOKEN;
  return h;
}

// ====== Custom metrics ======
const ttfb     = new Trend('llm_ttfb_ms', true);
const recvTime = new Trend('llm_receiving_ms', true);
const totalDur = new Trend('llm_total_ms', true);
const queueDur = new Trend('llm_queue_ms', true);
// tokens per second is NOT a time metric; don't mark as time
const tokRate  = new Trend('llm_tokens_per_sec');
const errors   = new Counter('llm_errors');

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

// ====== Scenarios ======
const minutes = (n) => `${n}m`;

const scenarios = {
  meta_smoke: {
    executor: 'ramping-arrival-rate',
    startRate: 1,
    timeUnit: '1s',
    preAllocatedVUs: 5,
    maxVUs: 20,
    stages: [
      { duration: minutes(1), target: 2 },
      { duration: minutes(DURATION_MINUTES - 2 > 0 ? DURATION_MINUTES - 2 : 1), target: 2 },
      { duration: minutes(1), target: 0 },
    ],
    exec: 'hitMeta',
    tags: { scenario: 'meta' },
  },

  // Non-stream short
  chat_nonstream_short: {
    executor: 'ramping-arrival-rate',
    startRate: Math.min(1, TARGET_NONSTREAM_RPS),
    timeUnit: '1s',
    preAllocatedVUs: 30,
    maxVUs: 200,
    stages: [
      { duration: minutes(1), target: TARGET_NONSTREAM_RPS },
      { duration: minutes(DURATION_MINUTES - 2 > 0 ? DURATION_MINUTES - 2 : 1), target: TARGET_NONSTREAM_RPS },
      { duration: minutes(1), target: 0 },
    ],
    exec: 'chatNonStreamShort',
    tags: { scenario: 'chat_nonstream', prompt: 'short' },
  },

  // Non-stream COT
  chat_nonstream_cot: {
    executor: 'ramping-arrival-rate',
    startRate: Math.min(1, TARGET_NONSTREAM_RPS),
    timeUnit: '1s',
    preAllocatedVUs: 30,
    maxVUs: 200,
    stages: [
      { duration: minutes(1), target: TARGET_NONSTREAM_RPS },
      { duration: minutes(DURATION_MINUTES - 2 > 0 ? DURATION_MINUTES - 2 : 1), target: TARGET_NONSTREAM_RPS },
      { duration: minutes(1), target: 0 },
    ],
    exec: 'chatNonStreamCOT',
    tags: { scenario: 'chat_nonstream', prompt: 'cot' },
  },

  ...(TARGET_STREAM_RPS > 0
    ? {
        chat_stream_short: {
          executor: 'ramping-arrival-rate',
          startRate: Math.min(1, TARGET_STREAM_RPS),
          timeUnit: '1s',
          preAllocatedVUs: 30,
          maxVUs: 200,
          stages: [
            { duration: minutes(1), target: TARGET_STREAM_RPS },
            { duration: minutes(DURATION_MINUTES - 2 > 0 ? DURATION_MINUTES - 2 : 1), target: TARGET_STREAM_RPS },
            { duration: minutes(1), target: 0 },
          ],
          exec: 'chatStreamShort',
          tags: { scenario: 'chat_stream', prompt: 'short' },
        },
        chat_stream_cot: {
          executor: 'ramping-arrival-rate',
          startRate: Math.min(1, TARGET_STREAM_RPS),
          timeUnit: '1s',
          preAllocatedVUs: 30,
          maxVUs: 200,
          stages: [
            { duration: minutes(1), target: TARGET_STREAM_RPS },
            { duration: minutes(DURATION_MINUTES - 2 > 0 ? DURATION_MINUTES - 2 : 1), target: TARGET_STREAM_RPS },
            { duration: minutes(1), target: 0 },
          ],
          exec: 'chatStreamCOT',
          tags: { scenario: 'chat_stream', prompt: 'cot' },
        },
      }
    : {}),
};

export const options = {
  scenarios,
  thresholds: {
    'http_req_failed': ['rate<0.02'],
    'llm_ttfb_ms{scenario:chat_stream,status:200,prompt:short}': ['p(95)<1000'],
    'llm_ttfb_ms{scenario:chat_stream,status:200,prompt:cot}':   ['p(95)<3000'],
  },
  discardResponseBodies: false,
};

// ====== Helpers ======
function payload(messages, stream) {
  return JSON.stringify({ model: MODEL, stream, messages });
}

function shortPrompt() {
  return [
    { role: 'system', content: 'You are a helpful assistant. Answer concisely.' },
    { role: 'user', content: 'A train travels 60 miles in 1.5 hours. What is its average speed?' },
  ];
}

function cotPrompt() {
  return [
    {
      role: 'system',
      content:
        'You are an expert reasoning assistant. Before final answer, think step-by-step and write reasoning in "Thought process". Then give final answer in "Final answer".',
    },
    { role: 'user', content: 'A train travels 60 miles in 1.5 hours. What is its average speed?' },
  ];
}

// ====== Exec functions ======
export function hitMeta() {
  let res = http.get(`${BASE}/v1/version`, { headers: buildHeaders() });
  recordTimings(res, 'meta', 'na');
  let ok1 = check(res, { 'version 200': (r) => r.status === 200 });

  res = http.get(`${BASE}/v1/models`, { headers: buildHeaders() });
  recordTimings(res, 'meta', 'na');
  let ok2 = check(res, { 'models 200': (r) => r.status === 200 });

  if (!(ok1 && ok2)) errors.add(1);
  sleep(1);
}

function doChat(messages, stream, scenario, promptType) {
  const body = payload(messages, stream);
  const res = http.post(`${BASE}/v1/chat/completions`, body, { headers: buildHeaders() });

  recordTimings(res, scenario, promptType);

  const ok = check(res, {
    'status 200': (r) => r.status === 200,
  });

  if (!ok) errors.add(1);
  sleep(1);
}

// Non-stream short vs COT
export function chatNonStreamShort() { doChat(shortPrompt(), false, 'chat_nonstream', 'short'); }
export function chatNonStreamCOT()   { doChat(cotPrompt(),   false, 'chat_nonstream', 'cot'); }

// Stream short vs COT
export function chatStreamShort()    { doChat(shortPrompt(), true, 'chat_stream', 'short'); }
export function chatStreamCOT()      { doChat(cotPrompt(),   true, 'chat_stream', 'cot'); }
