import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 10,
  duration: "5m",
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export default function () {
  const headers = { "Content-Type": "application/json" };

  // Create PR
  let payload = JSON.stringify({
    pull_request_id: `pr-${__ITER % 1000}`,
    pull_request_name: "performance",
    author_id: "u1",
  });
  let res = http.post(`${BASE_URL}/pullRequest/create`, payload, { headers });
  check(res, {
    "create status is 201 or 409": (r) => r.status === 201 || r.status === 409,
  });

  // Reassign reviewer
  payload = JSON.stringify({
    pull_request_id: "pr-0",
    old_user_id: "u2",
  });
  res = http.post(`${BASE_URL}/pullRequest/reassign`, payload, { headers });
  check(res, {
    "reassign status is 200 or 409": (r) => r.status === 200 || r.status === 409,
  });

  // Get reviewer queue
  res = http.get(`${BASE_URL}/users/getReview?user_id=u2`);
  check(res, {
    "getReview status is 200": (r) => r.status === 200,
  });

  sleep(0.2);
}
