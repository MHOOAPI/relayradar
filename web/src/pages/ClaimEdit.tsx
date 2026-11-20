import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api';
import type { ClaimedStation, ClaimEdit } from '../types';
import { Spinner } from '../components/ui';
import { useToast } from '../lib/toast';

export function ClaimEditPage() {
  const { token = '' } = useParams();
  const [station, setStation] = useState<ClaimedStation | null>(null);
  const [edits, setEdits] = useState<ClaimEdit[]>([]);
  const [editLimit, setEditLimit] = useState(20);
  const [tagline, setTagline] = useState('');
  const [refund, setRefund] = useState('');
  const [invoice, setInvoice] = useState(false);
  const [minTopup, setMinTopup] = useState('0');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const { push } = useToast();

  useEffect(() => {
    api.claimedStation(token)
      .then((s) => {
        setStation(s);
        setTagline(s.tagline);
        setRefund(s.refundPolicy);
        setInvoice(s.invoice);
        setMinTopup(String(s.minTopupCny || 0));
      })
      .catch((e: Error) => setError(e.message));
    api.claimEdits(token, editLimit).then((r) => setEdits(r.items)).catch(() => setEdits([]));
  }, [token, editLimit]);

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      const r = await api.updateClaimedStation(token, { tagline: tagline.trim(), refundPolicy: refund.trim(), invoice, minTopupCny: Number(minTopup) || 0 });
      push(r.pending ? '已提交审核，后台通过后生效' : '站点资料已更新');
    } catch (err) {
      push((err as Error).message, 'err');
    } finally {
      setBusy(false);
    }
  };

  if (error) {
    return <main className="wrap"><div className="error-box">{error} <Link className="link-btn" to="/submit">返回认领</Link></div></main>;
  }
  if (!station) return <main className="wrap"><div className="loading-page"><Spinner /> 读取认领信息…</div></main>;

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/submit">认领</Link><span>/</span><span className="dim">{station.slug}</span></div>
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />CLAIMED PROFILE</span>
          <h1>{station.name} · 资料维护</h1>
          <p className="sub">认证站长可维护简介、退款说明、发票与最低充值。提交后需后台审核，探针、评分与用户评价不可修改。</p>
        </div>
        <Link className="btn btn-ghost btn-sm" to={`/stations/${station.slug}`}>查看档案 →</Link>
      </div>

      {station.reviewStatus && (
        <div className={`review-banner ${station.reviewStatus}`}>
          <b>
            最近审核：
            {station.reviewStatus === 'pending' ? '待审核' : station.reviewStatus === 'approved' ? '已通过' : '未通过'}
          </b>
          {station.reviewNote && <span>审核意见：{station.reviewNote}</span>}
        </div>
      )}

      <section className="panel pad">
        <form className="auth-form" onSubmit={save}>
          <label>
            <span>一句话简介</span>
            <input value={tagline} onChange={(e) => setTagline(e.target.value)} maxLength={80} placeholder="例如：官转聚合 · 企业合规主体" />
          </label>
          <label>
            <span>退款说明</span>
            <input value={refund} onChange={(e) => setRefund(e.target.value)} maxLength={120} placeholder="例如：余额 7 天原路退" />
          </label>
          <label>
            <span>最低充值（CNY）</span>
            <input type="number" min={0} step={1} value={minTopup} onChange={(e) => setMinTopup(e.target.value)} />
          </label>
          <label className="check-inline">
            <input type="checkbox" checked={invoice} onChange={(e) => setInvoice(e.target.checked)} />
            支持开具发票
          </label>
          <button className="btn btn-primary" disabled={busy}>{busy ? '保存中…' : '保存资料'}</button>
        </form>
      </section>
      <section className="panel pad">
        <h2 style={{ marginBottom: 10 }}>提交记录</h2>
        {edits.length === 0 && <p className="fine">还没有提交过修改。</p>}
        <div className="claim-history">
          {edits.map((e) => (
            <div key={e.id} className="claim-history-row">
              <span className={`status-pill ${e.status === 'pending' ? 'degraded' : e.status === 'approved' ? 'operational' : 'outage'}`}>
                {e.status === 'pending' ? '待审核' : e.status === 'approved' ? '已通过' : '未通过'}
              </span>
              <span className="mono dim">{new Date(e.createdAt).toLocaleString('zh-CN', { hour12: false })}</span>
              <span className="fine">{e.tagline || '—'}</span>
              {e.reviewNote && <span className="fine">意见：{e.reviewNote}</span>}
            </div>
          ))}
        </div>
        {edits.length >= editLimit && editLimit < 100 && (
          <button className="btn btn-ghost btn-sm" style={{ marginTop: 10 }} onClick={() => setEditLimit((n) => Math.min(100, n + 20))}>
            加载更多
          </button>
        )}
      </section>
    </main>
  );
}
