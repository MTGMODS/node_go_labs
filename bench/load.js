import http from "k6/http";
import { check } from "k6";

const baseURL = __ENV.BASE_URL || "http://node:3000";
const endpoint = __ENV.ENDPOINT || "/io?delay_ms=100";

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || "20s",
  summaryTrendStats: ["avg", "p(95)", "p(99)", "max"],
  thresholds: {
    http_req_failed: ["rate==0"],
  },
};

export default function () {
  const response = http.get(`${baseURL}${endpoint}`, {
    tags: { workload: __ENV.WORKLOAD || "unknown" },
    timeout: "120s",
  });

  check(response, {
    "status is 200": (result) => result.status === 200,
  });
}
