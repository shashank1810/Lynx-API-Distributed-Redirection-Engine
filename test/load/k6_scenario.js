import http from "k6/http";
import { check, sleep, group } from "k6";
import { Rate, Trend } from "k6/metrics";

// ── Custom Metrics ──────────────────────────────────────
const shortenErrors = new Rate("shorten_errors");
const resolveErrors = new Rate("resolve_errors");
const shortenDuration = new Trend("shorten_duration", true);
const resolveDuration = new Trend("resolve_duration", true);

// ── Test Configuration ──────────────────────────────────
export const options = {
  scenarios: {
    // Scenario 1: Ramp-up smoke test
    smoke: {
      executor: "ramping-vus",
      startVUs: 1,
      stages: [
        { duration: "30s", target: 10 },
        { duration: "1m", target: 10 },
        { duration: "10s", target: 0 },
      ],
      gracefulRampDown: "5s",
      exec: "smokeTest",
    },

    // Scenario 2: Sustained load test
    load: {
      executor: "constant-arrival-rate",
      rate: 500,           // 500 requests per second
      timeUnit: "1s",
      duration: "2m",
      preAllocatedVUs: 100,
      maxVUs: 200,
      exec: "loadTest",
      startTime: "2m",    // start after smoke test
    },

    // Scenario 3: Spike test
    spike: {
      executor: "ramping-arrival-rate",
      startRate: 50,
      timeUnit: "1s",
      stages: [
        { duration: "10s", target: 50 },
        { duration: "10s", target: 2000 },  // spike to 2000 RPS
        { duration: "30s", target: 2000 },
        { duration: "10s", target: 50 },
      ],
      preAllocatedVUs: 200,
      maxVUs: 500,
      exec: "loadTest",
      startTime: "5m",    // start after load test
    },
  },

  thresholds: {
    http_req_duration: ["p(95)<200", "p(99)<500"],     // 95th < 200ms, 99th < 500ms
    http_req_failed: ["rate<0.01"],                     // <1% failure rate
    shorten_errors: ["rate<0.01"],
    resolve_errors: ["rate<0.01"],
    shorten_duration: ["p(95)<300"],
    resolve_duration: ["p(95)<100"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

// ── Helpers ─────────────────────────────────────────────
function shortenURL(url) {
  const payload = JSON.stringify({ url: url });
  const params = {
    headers: { "Content-Type": "application/json" },
  };

  const res = http.post(`${BASE_URL}/api/v1/shorten`, payload, params);
  shortenDuration.add(res.timings.duration);

  const success = check(res, {
    "shorten: status is 201": (r) => r.status === 201,
    "shorten: has short_code": (r) => {
      const body = r.json();
      return body && body.short_code && body.short_code.length > 0;
    },
  });

  shortenErrors.add(!success);

  if (res.status === 201) {
    return res.json().short_code;
  }
  return null;
}

function resolveURL(code) {
  const res = http.get(`${BASE_URL}/${code}`, {
    redirects: 0, // don't follow redirects — we want to inspect the 301
  });
  resolveDuration.add(res.timings.duration);

  const success = check(res, {
    "resolve: status is 301": (r) => r.status === 301,
    "resolve: has Location header": (r) =>
      r.headers["Location"] !== undefined,
  });

  resolveErrors.add(!success);
  return success;
}

// ── Smoke Test Scenario ────────────────────────────────
export function smokeTest() {
  group("Smoke: Shorten + Resolve", () => {
    const code = shortenURL(`https://example.com/page/${Date.now()}`);
    if (code) {
      sleep(0.1);
      resolveURL(code);
    }
  });

  group("Smoke: Health Check", () => {
    const res = http.get(`${BASE_URL}/healthz`);
    check(res, {
      "healthz: status is 200": (r) => r.status === 200,
    });
  });

  sleep(1);
}

// ── Load Test Scenario ─────────────────────────────────
export function loadTest() {
  // 70% reads, 30% writes (realistic URL shortener workload)
  if (Math.random() < 0.3) {
    // Write: create a new short URL
    shortenURL(`https://example.com/load/${Date.now()}/${Math.random()}`);
  } else {
    // Read: shorten then immediately resolve
    const code = shortenURL(
      `https://example.com/resolve/${Date.now()}/${Math.random()}`
    );
    if (code) {
      resolveURL(code);
    }
  }
}
