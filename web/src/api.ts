import type {
  AdminSummary,
  AccountData,
  AlertSubscription,
  DeliveryLog,
  FreeKeyView,
  Incident,
  ModelProviderStatus,
  StatusResponse,
  AuditReport,
  AuditCheckDoc,
  Change,
  ClaimResponse,
  ClaimEdit,
  ClaimedStation,
  Meta,
  ModelSummary,
  ModelOption,
  Offer,
  PriceChange,
  Review,
  Station,
  StationDetail,
  StationReachabilityHistory,
  Submission,
  User,
  WindowRow,
} from './types';
import type { GuideDetail, GuideEN, GuideSummary } from './types/guide';

const BASE = '/api/v1';

// Slider-captcha token for sensitive writes. The server issues it after the
// puzzle is solved; requests simply carry it while it is valid.
let captchaToken = '';
export function setCaptchaToken(token: string) { captchaToken = token; }
export function clearCaptchaToken() { captchaToken = ''; }

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const isForm = typeof FormData !== 'undefined' && init?.body instanceof FormData;
  const res = await fetch(BASE + path, {
    headers: {
      Accept: 'application/json',
      ...(init?.body && !isForm ? { 'Content-Type': 'application/json' } : {}),
      ...(captchaToken ? { 'X-Captcha-Token': captchaToken } : {}),
    },
    ...init,
  });
  if (!res.ok) {
    let message = `请求失败 (HTTP ${res.status})`;
    try {
      const data = (await res.json()) as { error?: string };
      if (data.error) message = data.error;
    } catch {
      /* keep default */
    }
    throw new ApiError(res.status, message);
  }
  return (await res.json()) as T;
}

export const api = {
  meta: () => request<Meta>('/meta'),
  stations: () => request<{ items: Station[]; total: number }>('/stations'),
  station: (slug: string) => request<StationDetail>(`/stations/${encodeURIComponent(slug)}`),
  stationReachability: (slug: string) => request<StationReachabilityHistory>(`/stations/${encodeURIComponent(slug)}/reachability`),
  models: () => request<{ items: ModelSummary[]; total: number }>('/models'),
  model: (id: string) => request<{ model: ModelSummary; offers: Offer[]; priceChanges: PriceChange[] }>(`/models/${encodeURIComponent(id)}`),
  reviews: (slug: string, body: { author: string; rating: number; body: string }) =>
    request<Review>(`/stations/${encodeURIComponent(slug)}/reviews`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  audit: (body: { baseUrl: string; apiKey: string; model: string; lang?: string }) =>
    request<AuditReport>('/audit', { method: 'POST', body: JSON.stringify(body) }),
  auditModels: (baseUrl: string, apiKey: string) =>
    request<{ items: ModelOption[]; count: number }>('/audit/models', {
      method: 'POST',
      body: JSON.stringify({ baseUrl, apiKey }),
    }),
  outbound: (slug: string) =>
    request<{ ok: boolean; count30d: number }>('/outbound', {
      method: 'POST',
      body: JSON.stringify({ slug }),
    }),
  auditReport: (id: string) => request<AuditReport>(`/audit/${encodeURIComponent(id)}`),
  captcha: () =>
    request<{ disabled?: boolean; id: string; background: string; piece: string; width: number; height: number; pieceSize: number; pieceY: number; maxX: number }>('/captcha'),
  verifyCaptcha: (id: string, x: number) =>
    request<{ token: string }>(`/captcha/${encodeURIComponent(id)}/verify`, { method: 'POST', body: JSON.stringify({ x }) }),
  auditChecks: () => request<{ items: AuditCheckDoc[]; count: number }>('/audit/checks'),
  auditCheck: (key: string) => request<AuditCheckDoc>(`/audit/checks/${encodeURIComponent(key)}`),
  auditChecksEn: () => request<{ items: AuditCheckDoc[] }>('/audit/checks/en'),
  auditCheckEn: (key: string) => request<AuditCheckDoc>(`/audit/checks/en/${encodeURIComponent(key)}`),
  submitStation: (body: { name: string; domain: string; baseUrl: string; contact: string; note: string; payments: string[]; invoice: boolean; refundPolicy: string }) =>
    request<Submission>('/submissions', { method: 'POST', body: JSON.stringify(body) }),
  prefill: (baseUrl: string, apiKey: string) =>
    request<{ domain: string; baseUrl: string; models: string[]; modelsOk: boolean; pricing: { name: string; priceUsd: number }[]; pricingOk: boolean; textPricing: { name: string; inputCny: number; outputCny: number }[]; textPricingOk: boolean }>(
      '/submissions/prefill',
      { method: 'POST', body: JSON.stringify({ baseUrl, apiKey }) },
    ),
  claim: (body: { slug: string; email: string }) =>
    request<ClaimResponse>('/claims', { method: 'POST', body: JSON.stringify(body) }),
  claimedStation: (token: string) => request<ClaimedStation>(`/claims/${encodeURIComponent(token)}/station`),
  claimEdits: (token: string, limit = 20) => request<{ items: ClaimEdit[] }>(`/claims/${encodeURIComponent(token)}/edits?limit=${limit}`),
  updateClaimedStation: (token: string, body: { tagline: string; refundPolicy: string; invoice: boolean; minTopupCny: number }) =>
    request<{ ok: boolean; pending?: boolean; id?: number }>(`/claims/${encodeURIComponent(token)}/station`, { method: 'PATCH', body: JSON.stringify(body) }),
  adminClaimEdits: (all = false) => request<{ items: ClaimEdit[] }>(`/admin/claim-edits${all ? '?all=1' : ''}`),
  adminProbeConfig: () => request<{ config: string; exists: boolean }>('/admin/probe-config'),
  saveAdminProbeConfig: (config: string) => request<{ ok: boolean; targets: number }>('/admin/probe-config', { method: 'POST', body: JSON.stringify({ config }) }),
  approveClaimEdit: (id: number, note = '') => request<{ ok: boolean }>(`/admin/claim-edits/${id}/approve`, { method: 'POST', body: JSON.stringify({ note }) }),
  rejectClaimEdit: (id: number, note = '') => request<{ ok: boolean }>(`/admin/claim-edits/${id}/reject`, { method: 'POST', body: JSON.stringify({ note }) }),
  verifyClaim: (token: string) =>
    request<{ verified: boolean; station: string }>(`/claims/${encodeURIComponent(token)}/verify`, {
      method: 'POST',
      body: JSON.stringify({}),
    }),
  changes: (station?: string, kind?: string, lang?: string) => {
    const q = new URLSearchParams();
    if (station) q.set('station', station);
    if (kind && kind !== 'all') q.set('kind', kind);
    if (lang) q.set('lang', lang);
    const qs = q.toString();
    return request<{ items: Change[]; total: number }>(`/changes${qs ? `?${qs}` : ''}`);
  },
  guides: () => request<{ items: GuideSummary[] }>('/guides'),  guide: (slug: string) => request<GuideDetail>(`/guides/${encodeURIComponent(slug)}`),
  guidesEn: () => request<{ items: GuideEN[] }>('/guides/en'),
  guideEn: (slug: string) => request<GuideEN>(`/guides/en/${encodeURIComponent(slug)}`),
  subscribe: (body: { channel: 'webhook' | 'email'; url: string; email: string; kinds: string[]; stations: string[]; providers: string[] }) =>
    request<{ id: string; channel: string; url: string; email: string; secret: string; kinds: string[]; stations: string[]; providers: string[] }>('/subscriptions', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  status: () => request<StatusResponse>('/status'),
  modelStatus: (lang?: string) => request<{ providers: ModelProviderStatus[]; checkedAt: string }>(`/model-status${lang ? `?lang=${lang}` : ''}`),
  freeKeys: (q = '', includeInvalid = false) => {
    const p = new URLSearchParams();
    if (q) p.set('q', q);
    if (includeInvalid) p.set('include_invalid', '1');
    return request<{ items: FreeKeyView[]; count: number }>(`/free-keys${p.toString() ? `?${p}` : ''}`);
  },
  createFreeKey: (body: { baseUrl: string; apiKey: string; models?: string; note?: string; submitter?: string; website?: string }) =>
    request<{ id: number; status: string }>('/free-keys', { method: 'POST', body: JSON.stringify(body) }),
  claimFreeKey: (id: number) =>
    request<{ apiKey: string; baseUrl: string; models: string; note: string; warning: string }>(`/free-keys/${id}/claim`, { method: 'POST' }),
  pingSub: (id: string, token: string) =>
    request<{ ok: boolean; latencyMs: number }>(`/subscriptions/${encodeURIComponent(id)}/ping`, {
      method: 'POST',
      headers: { 'X-RelayRadar-Token': token },
    }),
  subDeliveries: (id: string, token: string) =>
    request<{ items: DeliveryLog[] }>(`/subscriptions/${encodeURIComponent(id)}/deliveries`, {
      headers: { 'X-RelayRadar-Token': token },
    }),
  replayDelivery: (id: string, token: string, logId: number) =>
    request<{ ok: boolean; status: number; latencyMs?: number; response?: string; error?: string }>(
      `/subscriptions/${encodeURIComponent(id)}/deliveries/${logId}/replay`,
      { method: 'POST', headers: { 'X-RelayRadar-Token': token } },
    ),
  getSub: (id: string, token: string) =>
    request<AlertSubscription>(`/subscriptions/${encodeURIComponent(id)}?token=${encodeURIComponent(token)}`),
  updateSub: (id: string, token: string, patch: { kinds?: string[]; enabled?: boolean }) =>
    request<AlertSubscription>(`/subscriptions/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      headers: { 'X-RelayRadar-Token': token },
      body: JSON.stringify(patch),
    }),
  deleteSub: (id: string, token: string) =>
    request<{ ok: boolean }>(`/subscriptions/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: { 'X-RelayRadar-Token': token },
    }),
  windowLeaderboard: (days: number) =>
    request<{ items: WindowRow[]; windowDays: number }>(`/leaderboards/window?days=${days}`),
  me: () => request<{ user: User }>('/auth/me'),
  account: () => request<AccountData>('/account'),
  accountSetSub: (id: string, enabled: boolean) =>
    request<{ ok: boolean }>(`/account/subscriptions/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify({ enabled }) }),
  accountDeleteSub: (id: string) =>
    request<{ ok: boolean }>(`/account/subscriptions/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  login: (email: string, password: string) =>
    request<{ user: User }>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  register: (email: string, password: string, name: string) =>
    request<{ user: User }>('/auth/register', { method: 'POST', body: JSON.stringify({ email, password, name }) }),
  logout: () => request<{ ok: boolean }>('/auth/logout', { method: 'POST' }),
  verifyEmail: (token: string) =>
    request<{ ok: boolean }>('/auth/verify', { method: 'POST', body: JSON.stringify({ token }) }),
  forgotPassword: (email: string) =>
    request<{ ok: boolean }>('/auth/forgot', { method: 'POST', body: JSON.stringify({ email }) }),
  resetPassword: (token: string, password: string) =>
    request<{ ok: boolean }>('/auth/reset', { method: 'POST', body: JSON.stringify({ token, password }) }),
  adminSummary: () => request<AdminSummary>('/admin/summary'),
  adminSubmissions: () => request<{ items: Submission[] }>('/admin/submissions'),
  adminUploadIcon: (slug: string, file: Blob) => {
    const body = new FormData();
    body.append('file', file, 'icon.png');
    return request<{ iconUrl: string }>(`/admin/stations/${encodeURIComponent(slug)}/icon`, { method: 'POST', body });
  },
  adminResetIcon: (slug: string) =>
    request<{ iconUrl: string }>(`/admin/stations/${encodeURIComponent(slug)}/icon`, { method: 'DELETE' }),
  adminSetSubmission: (id: number, status: string, terms?: { payments: string[]; invoice: boolean; refundPolicy: string }) =>
    request<{ ok: boolean }>(`/admin/submissions/${id}`, { method: 'PATCH', body: JSON.stringify({ status, ...(terms ?? {}) }) }),
  adminDeleteReview: (id: number) => request<{ ok: boolean }>(`/admin/reviews/${id}`, { method: 'DELETE' }),
  adminSubscriptions: () =>
    request<{ items: Array<{ id: string; channel: string; url: string; email: string; enabled: boolean; attempts: number; lastError: string; createdAt: string; lastSentAt: string }> }>('/admin/subscriptions'),
  adminUsers: () => request<{ items: User[] }>('/admin/users'),
  adminSetUserRole: (id: number, role: 'admin' | 'user') =>
    request<{ ok: boolean }>(`/admin/users/${id}`, { method: 'PATCH', body: JSON.stringify({ role }) }),
  adminReviews: () =>
    request<{ items: Array<{ id: number; stationSlug: string; stationName: string; author: string; rating: number; body: string; createdAt: string }> }>('/admin/reviews'),
  adminAudit: (limit = 100) =>
    request<{ items: Array<{ id: number; actorId: number; actorEmail: string; action: string; target: string; detail: string; ip: string; createdAt: string }> }>(`/admin/audit?limit=${limit}`),
  adminIncidents: () => request<{ items: Incident[] }>('/admin/incidents'),
  createIncident: (body: { title: string; component: string; body: string; status: string }) =>
    request<{ id: number }>('/admin/incidents', { method: 'POST', body: JSON.stringify(body) }),
  updateIncident: (id: number, body: { status: string; body: string }) =>
    request<{ ok: boolean }>(`/admin/incidents/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteIncident: (id: number) => request<{ ok: boolean }>(`/admin/incidents/${id}`, { method: 'DELETE' }),
};
