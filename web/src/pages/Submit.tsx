import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { ClaimResponse, Station } from '../types';
import { StationMark } from '../components/marks';

type Tab = 'submit' | 'claim';

export function SubmitPage() {
  const [params] = useSearchParams();
  const claimParam = params.get('claim') ?? '';
  const [tab, setTab] = useState<Tab>(claimParam ? 'claim' : 'submit');
  const [stations, setStations] = useState<Station[]>([]);

  const [form, setForm] = useState({ name: '', domain: '', baseUrl: '', contact: '', note: '', payments: [] as string[], invoice: false, refundPolicy: '' });
  const [apiKey, setApiKey] = useState('');
  const [prefilling, setPrefilling] = useState(false);
  const [prefillMsg, setPrefillMsg] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [slug, setSlug] = useState(claimParam);
  const [email, setEmail] = useState('');
  const [claim, setClaim] = useState<ClaimResponse | null>(null);
  const [claiming, setClaiming] = useState(false);
  const [verified, setVerified] = useState(false);

  useEffect(() => {
    api.stations().then((r) => {
      setStations(r.items);
      if (!claimParam && r.items[0]) setSlug(r.items[0].slug);
    }).catch(() => setStations([]));
  }, []);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const out = await api.submitStation(form);
      setSubmitted(out.id);
      setForm({ name: '', domain: '', baseUrl: '', contact: '', note: '', payments: [], invoice: false, refundPolicy: '' });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  const requestClaim = async (e: React.FormEvent) => {
    e.preventDefault();
    setClaiming(true);
    setError(null);
    try {
      const out = await api.claim({ slug, email });
      setClaim(out);
      setVerified(false);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setClaiming(false);
    }
  };

  const confirmClaim = async () => {
    if (!claim) return;
    setClaiming(true);
    setError(null);
    try {
      await api.verifyClaim(claim.claim.token);
      setVerified(true);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setClaiming(false);
    }
  };

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />CONTRIBUTE</span>
          <h1>提交与认领</h1>
          <p className="sub">提交新站点进入观察队列；站长可通过 DNS TXT 认领档案，补充倍率、退款与发票信息。</p>
        </div>
      </div>

      <div className="tabs">
        <button className={`tab ${tab === 'submit' ? 'on' : ''}`} onClick={() => setTab('submit')}>提交新站点</button>
        <button className={`tab ${tab === 'claim' ? 'on' : ''}`} onClick={() => { setTab('claim'); setError(null); }}>站长认领</button>
        <span className="tabs-note">审核周期约 3–7 天，不接受付费加急</span>
      </div>

      {error && <div className="error-box">{error}</div>}

      {tab === 'submit' && (
        <section className="panel form-panel">
          {submitted ? (
            <div className="success-box">
              <b>已提交，编号 #{submitted}</b>
              <p>我们会在 3–7 天内人工核验域名、公开报价页与退款条款；通过后会进入「观察期」并在榜单中标注。</p>
              <button className="btn btn-ghost btn-sm" onClick={() => setSubmitted(null)}>再提交一个</button>
            </div>
          ) : (
            <form className="form-grid" onSubmit={submit}>
              <label><span>站点名称 *</span><input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="例如 灵犀 API" required /></label>
              <label><span>主域名 *</span><input value={form.domain} onChange={(e) => setForm({ ...form, domain: e.target.value })} placeholder="lingxi-api.com" required /></label>
              <label><span>Base URL *</span><input value={form.baseUrl} onChange={(e) => setForm({ ...form, baseUrl: e.target.value })} placeholder="https://lingxi-api.com/v1" required /></label>
              <label><span>API Key（可选，用于自动填充）</span><input value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="sk-..." type="password" /></label>
              <div className="form-actions span-2">
                <button
                  type="button"
                  className="btn btn-sm"
                  disabled={!apiKey || prefilling}
                  onClick={async () => {
                    setPrefilling(true);
                    setPrefillMsg(null);
                    try {
                      const r = await api.prefill(form.baseUrl, apiKey);
                      setForm((f) => ({ ...f, domain: f.domain || r.domain, baseUrl: f.baseUrl || r.baseUrl }));
                      setPrefillMsg(`已拉取 ${r.models.length} 个模型${r.pricingOk ? `、${r.pricing.length} 条按次价` : ''}${r.textPricingOk ? `、${r.textPricing.length} 条文本价` : ''}`);
                    } catch (e) {
                      setPrefillMsg((e as Error).message);
                    } finally {
                      setPrefilling(false);
                    }
                  }}
                >
                  {prefilling ? '拉取中…' : '自动填充'}
                </button>
                {prefillMsg && <span className="fine">{prefillMsg}</span>}
              </div>
              <label><span>联系方式 *</span><input value={form.contact} onChange={(e) => setForm({ ...form, contact: e.target.value })} placeholder="邮箱 / Telegram" required /></label>
              <label className="span-2"><span>备注</span><textarea rows={3} value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} placeholder="倍率、分组、退款政策、可否开票等" maxLength={500} /></label>
              <label className="span-2"><span>支付方式</span>
                <div className="pay-picks">
                  {['支付宝', '微信', 'USDT', 'Stripe', '信用卡'].map((p) => (
                    <label key={p} className="pay-pick">
                      <input
                        type="checkbox"
                        checked={form.payments.includes(p)}
                        onChange={(e) => setForm({ ...form, payments: e.target.checked ? [...form.payments, p] : form.payments.filter((x) => x !== p) })}
                      />
                      {p}
                    </label>
                  ))}
                </div>
              </label>
              <label><span>是否支持开票</span><input type="checkbox" checked={form.invoice} onChange={(e) => setForm({ ...form, invoice: e.target.checked })} /></label>
              <label className="span-2"><span>退款政策</span><input value={form.refundPolicy} onChange={(e) => setForm({ ...form, refundPolicy: e.target.value })} placeholder="例如 7 天未用可退 / 按量可退" maxLength={200} /></label>
              <div className="form-actions span-2">
                <button className="btn btn-primary" disabled={submitting}>{submitting ? '提交中…' : '提交审核'}</button>
              </div>
            </form>
          )}
        </section>
      )}

      {tab === 'claim' && (
        <section className="panel form-panel">
          {verified ? (
            <div className="success-box">
              <b>认领成功</b>
              <p>该站点已标记为「✓ 已认领」，你可以在档案页补充商务条款。排名与探针数据不会被修改。</p>
              <p><Link className="btn btn-primary btn-sm" to={`/claim/${claim?.claim.token ?? ''}`}>编辑站点资料 →</Link></p>
              <Link className="btn btn-primary btn-sm" to={`/stations/${slug}`}>查看档案</Link>
            </div>
          ) : claim ? (
            <div className="claim-box">
              <h2>在 DNS 添加这条 TXT 记录</h2>
              <div className="dns-row">
                <span>类型</span><code className="mono">{claim.dns.type}</code>
              </div>
              <div className="dns-row">
                <span>主机记录</span><code className="mono">{claim.dns.name}</code>
              </div>
              <div className="dns-row">
                <span>记录值</span><code className="mono accent">{claim.dns.value}</code>
              </div>
              <p className="fine">也可以用根目录文件验证：<code className="mono">/.well-known/relayradar.txt</code> 内容为同一令牌。DNS 生效可能需要几分钟。</p>
              <div className="form-actions">
                <button className="btn btn-ghost btn-sm" onClick={() => setClaim(null)}>返回修改</button>
                <button className="btn btn-primary" onClick={confirmClaim} disabled={claiming}>
                  {claiming ? '校验中…' : '我已配置，立即校验'}
                </button>
              </div>
            </div>
          ) : (
            <form className="form-grid" onSubmit={requestClaim}>
              <label className="span-2">
                <span>选择要认领的站点</span>
                <select value={slug} onChange={(e) => setSlug(e.target.value)}>
                  {stations.map((s) => (
                    <option key={s.slug} value={s.slug}>{s.name}（{s.domain}）{s.claimed ? ' · 已认领' : ''}</option>
                  ))}
                </select>
              </label>
              <label className="span-2"><span>站长邮箱 *</span><input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="ops@example.com" required /></label>
              <div className="claim-preview span-2">
                {stations.filter((s) => s.slug === slug).map((s) => (
                  <span key={s.slug} className="claim-preview-inner">
                    <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={26} />
                    <b>{s.name}</b>
                    <span className="mono dim">{s.domain}</span>
                  </span>
                ))}
              </div>
              <div className="form-actions span-2">
                <span className="fine">认领只能修改商务条款与官方说明，不能修改探针、评分与用户评价。</span>
                <button className="btn btn-primary" disabled={claiming || stations.length === 0}>
                  {claiming ? '生成中…' : '生成验证令牌'}
                </button>
              </div>
            </form>
          )}
        </section>
      )}
    </main>
  );
}
