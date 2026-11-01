import http from 'k6/http';
import { check } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  vus: 50,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<300'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const res = http.post('http://localhost:8080/transfers', JSON.stringify({
    from_account_id: __ENV.FROM,
    to_account_id: __ENV.TO,
    amount: 1,
    idempotency_key: uuidv4(),
    description: 'load test',
  }), {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${__ENV.TOKEN}` },
  });
  check(res, { 'status 200': (r) => r.status === 200 });
}
