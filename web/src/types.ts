export type PurityTier = 'official' | 'mixed' | 'counterfeit' | 'unknown';
export type CheckStatus = 'pass' | 'warn' | 'fail';

export interface Station {
  slug: string;
  name: string;
  domain: string;
  baseUrl: string;
  iconUrl: string;
  tagline: string;
  taglineEn: string;
  officialVendor: boolean;
  rank: number;
  rankScore: number;
  scoreMargin: number;
  availability: number;
  speed: number;
  consistency: number;
  samples: number;
  scoredGroups: number;
  uptimePct: number;
  purity: number;
  purityTier: PurityTier;
  purityMeasured: boolean;
  modelsCount: number;
  modelIds?: string[];
  reachability?: { status: string; latencyMs: number; checkedAt: string };
  reviewsCount: number;
  ratingAvg: number;
  groupsCount: number;
  fromPriceCny: number;
  minTopupCny: number;
  payments: string[];
  invoice: boolean;
  refundPolicy: string;
  claimed: boolean;
  services: string[];
  uptime: number[];
  ranked: boolean;
  newDomain: boolean;
  listedAt: string;
  lastProbeAt: string;
}

export interface ModelSpec {
  id: string;
  vendor: string;
  vendorName: string;
  name: string;
  service: string;
  category?: 'text' | 'image' | 'video';
  contextWindow: number;
  inputUsd: number;
  outputUsd: number;
  releasedAt: string;
  codingScore: number;
  reasoningScore: number;
  freeTier: boolean;
  freeNote: string;
  summary: string;
}

export interface ModelSummary extends ModelSpec {
  offersCount: number;
  minInputCny: number;
  minOutputCny: number;
  bestStation: string;
  license: string;
  summaryEn: string;
  // Supply signals and scores computed from offers.
  priceLowCny: number;
  priceHighCny: number;
  priceMedianCny: number;
  avgAvailability: number;
  purityOfficial: number;
  purityMixed: number;
  purityUnknown: number;
  purityCounterfeit: number;
  priceRatio: number;
  priceScore: number;
  coverageScore: number;
  availabilityScore: number;
  purityScore: number;
  contextScore: number;
  relayScore: number;
}

export interface Group {
  slug: string;
  name: string;
  service: string;
  multiplier: number;
  status: 'ok' | 'slow' | 'down' | 'unknown';
  latencyMs: number;
  availability: number;
  speed: number;
  consistency: number;
  quality: number;
  samples: number;
  scored: boolean;
  purity: number;
  purityTier: PurityTier;
  modelCount: number;
  lastProbeAt: string;
}

export interface StationModelRow {
  model: ModelSpec;
  bestInputCny: number;
  bestOutputCny: number;
  bestCacheCny: number;
  cheapestGroup: string;
  groups: string[];
}

export interface Review {
  id: number;
  author: string;
  rating: number;
  body: string;
  verified: boolean;
  createdAt: string;
}

export interface StationDetail {
  station: Station;
  groups: Group[];
  models: StationModelRow[];
  reviews: Review[];
  business: {
    minTopupCny: number;
    payments: string[];
    invoice: boolean;
    refundPolicy: string;
    listedAt: string;
    claimed: boolean;
  };
  history: HistoryPoint[];
  outbound30d: number;
}

export interface HistoryPoint {
  date: string;
  score: number;
  availability: number;
  speed: number;
  consistency: number;
  uptime: number;
}

export interface ModelOption {
  id: string;
  ownedBy?: string;
}

export interface AuditCheckDoc {
  key: string;
  label: string;
  short: string;
  why: string;
  how: string;
  pass: string;
  warn: string;
  fail: string;
  limit: string;
}

export interface AlertSubscription {
  id: string;
  channel: 'webhook' | 'email';
  url: string;
  email: string;
  kinds: string[];
  stations: string[];
  providers: string[];
  enabled: boolean;
  attempts: number;
  lastError: string;
  nextAttemptAt?: string;
  createdAt: string;
  lastSentAt: string;
}

export interface ClaimEdit {
  id: number;
  stationSlug: string;
  stationName: string;
  tagline: string;
  refundPolicy: string;
  invoice: boolean;
  minTopupCny: number;
  status: string;
  reviewNote: string;
  createdAt: string;
}

export interface ClaimedStation {
  slug: string;
  name: string;
  tagline: string;
  refundPolicy: string;
  invoice: boolean;
  minTopupCny: number;
  reviewStatus: string;
  reviewNote: string;
}

export interface ModelProviderStatus {
  key: string;
  name: string;
  status: 'operational' | 'degraded' | 'outage' | 'unknown';
  description: string;
  statusUrl: string;
  checkedVia: 'statuspage' | 'probe' | string;
  checkedAt: string;
  components?: Array<{ name: string; status: string }>;
  incidents?: Array<{
    name: string;
    status: string;
    impact: string;
    url: string;
    created: string;
    startedAt?: string;
    resolvedAt?: string;
    components?: string[];
    updates?: Array<{ status: string; body: string; createdAt: string }>;
  }>;
  history?: Array<{ date: string; status: 'operational' | 'degraded' | 'outage' | 'unknown'; uptime: number }>;
}

export interface StatusComponent {
  key: string;
  name: string;
  status: 'operational' | 'degraded' | 'outage' | 'unknown';
  detail: string;
}

export interface StatusResponse {
  status: 'operational' | 'degraded' | 'outage';
  checkedAt: string;
  startedAt: string;
  uptimeSeconds: number;
  version: string;
  components: StatusComponent[];
  history: StatusHistory[];
  incidents: Incident[];
}

export interface StatusHistoryDay {
  date: string;
  status: 'operational' | 'degraded' | 'outage' | 'unknown';
  uptime: number;
}

export interface StatusHistory {
  key: string;
  days: StatusHistoryDay[];
}

export interface StationReachabilityHistory {
  slug: string;
  days: StatusHistoryDay[];
}

export interface Incident {
  id: number;
  title: string;
  status: 'investigating' | 'identified' | 'monitoring' | 'resolved';
  component: string;
  body: string;
  startedAt: string;
  resolvedAt?: string;
  createdAt: string;
}

export interface FreeKeyView {
  id: number;
  provider: string;
  baseUrl: string;
  keyMasked: string;
  models: string;
  note: string;
  submitter: string;
  status: 'available' | 'invalid' | 'unknown';
  claims: number;
  createdAt: string;
}

export interface DeliveryLog {
  id: number;
  subscriptionId: string;
  event: string;
  status: number;
  ok: boolean;
  durationMs: number;
  response: string;
  createdAt: string;
}

export interface PriceChange {  stationSlug: string;
  stationName: string;
  groupName: string;
  oldInputCny: number;
  newInputCny: number;
  deltaPct: number;
  at: string;
}

export interface Offer {
  stationSlug: string;
  stationName: string;
  groupSlug: string;
  groupName: string;
  multiplier: number;
  inputCny: number;
  outputCny: number;
  cacheReadCny: number;
  stationRank: number;
  purityTier: PurityTier;
  availability: number;
  claimed: boolean;
  iconUrl?: string;
}

export interface Meta {
  stations: number;
  groups: number;
  models: number;
  probes: number;
  dataUpdatedAt: string;
  demo: boolean;
  auditEnabled: boolean;
  methodologyVersion: string;
}

export interface Check {
  key: string;
  label: string;
  status: CheckStatus;
  detail: string;
}

export interface Submission {
  id: number;
  name: string;
  domain: string;
  baseUrl: string;
  contact: string;
  note: string;
  payments?: string[];
  invoice?: boolean;
  refundPolicy?: string;
  status: string;
  createdAt: string;
}

export interface Claim {
  id: number;
  email: string;
  token: string;
  verified: boolean;
  createdAt: string;
}

export interface ClaimResponse {
  claim: Claim;
  dns: { type: string; name: string; value: string };
}

export interface Change {
  id: string;
  kind: 'new_station' | 'purity_alert' | 'availability_drop' | 'price_change';
  severity: 'info' | 'warn' | 'critical';
  stationSlug: string;
  stationName: string;
  title: string;
  detail: string;
  at: string;
}

export interface AuditReport {
  id: string;
  createdAt: string;
  targetUrl: string;
  endpoint: string;
  model: string;
  passed: boolean;
  ttftMs: number;
  durationMs: number;
  promptTokens: number;
  completionTokens: number;
  cachedTokens: number;
  echoModel: string;
  checks: Check[];
}

export interface User {
  id: number;
  email: string;
  name: string;
  role: 'user' | 'admin';
  emailVerified: boolean;
  createdAt: string;
}

export interface AdminSummary {  counts: Record<string, number>;
  updatedAt: string;
  subscriptions: number;
  submissions: number;
  pendingSubmissions: number;
  failingSubscriptions: number;
  claimedStations: number;
  pendingClaimEdits: number;
  staleProbes: number;
}

export interface WindowRow {
  slug: string;
  name: string;
  average: number;
  current: number;
  delta: number;
  uptime: number;
  days: number;
}

export interface AccountData {
  user: User;
  stats: { subscriptions: number; active: number; submissions: number; pending: number };
  subscriptions: AlertSubscription[];
  submissions: Submission[];
}
export interface SearchStation {
  slug: string;
  name: string;
  domain: string;
  iconUrl: string;
  rank: number;
  rankScore: number;
  fromPriceCny: number;
  purityTier: string;
  officialVendor: boolean;
}

export interface SearchModel {
  id: string;
  name: string;
  vendor: string;
  vendorName: string;
  category: string;
  minInputCny: number;
  offersCount: number;
}