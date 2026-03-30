// k6：规则快路径冒烟（SYS-PERF-01）；setup 预置精确规则，避免 404/无命中。
import http from "k6/http";
import { check, sleep } from "k6";

const BASE = __ENV.BASE_URL || "http://127.0.0.1:8080";
const STRICT = __ENV.STRICT_PERF === "1";

export const options = {
  vus: 5,
  duration: "10s",
  thresholds: {
    // STRICT_PERF=1 时按 NFR-P01 口径收紧（15ms）；默认 50ms 冒烟防 CI/本机抖动误杀
    http_req_duration: [STRICT ? "p(99)<15" : "p(99)<50"],
  },
};

export function setup() {
  const payload = JSON.stringify({
    key: "k6-smoke-key",
    dat: "k6-dat",
  });
  const res = http.post(`${BASE}/v1/admin/rules/exact`, payload, {
    headers: { "Content-Type": "application/json" },
  });
  if (res.status !== 201 && res.status !== 409) {
    throw new Error(`setup: expect 201/409, got ${res.status} body=${String(res.body).slice(0, 200)}`);
  }
  return { base: BASE };
}

export default function (data) {
  const base = (data && data.base) || BASE;
  const res = http.post(
    `${base}/v1/qa`,
    JSON.stringify({ query: "k6-smoke", key: "k6-smoke-key" }),
    {
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
      },
      timeout: "5s",
    }
  );
  check(res, { "2xx": (r) => r.status === 200 });
  sleep(0.05);
}
