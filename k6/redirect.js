// 2 000 переходов в секунду с постоянной частотой.
// 20% запросов — одна «горячая» ссылка (рассылка), 1% — несуществующие коды.
import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE || 'http://redirect:8080';
const RATE = Number(__ENV.RATE || 2000);
const DURATION = __ENV.DURATION || '60s';

export const options = {
  maxRedirects: 0,
  discardResponseBodies: true,
  scenarios: {
    redirects: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 200,
      maxVUs: 1000,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.001'],
    http_req_duration: ['p(99)<200'],
  },
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function setup() {
  return { codes: http.get(`${BASE}/_codes?n=100000`, { responseType: 'text' }).json() };
}

export default function (data) {
  const r = Math.random();
  let code, expected;
  if (r < 0.01) {
    code = 'zzzzzzz'; expected = 404;
  } else if (r < 0.21) {
    code = data.codes[0]; expected = 302;
  } else {
    code = data.codes[Math.floor(Math.random() * data.codes.length)]; expected = 302;
  }
  const res = http.get(`${BASE}/${code}`, {
    tags: { name: 'redirect' },
    responseCallback: http.expectedStatuses(302, 404),
  });
  check(res, { 'ожидаемый статус': (x) => x.status === expected });
}
