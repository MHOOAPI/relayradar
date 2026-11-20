import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { ModelSummary } from '../../types';
import { fmtCny, Spinner } from '../../components/ui';

const FREE_NOTE_EN: Record<string, string> = {
  'gemini-4-hydrogen': 'AI Studio free tier, 1,000 requests/day (billing account required)',
  'deepseek-v4-1-flash': 'New accounts receive 5M tokens, valid for 30 days',
  'qwen4-max': 'Alibaba Model Studio daily free quota of 1M tokens',
};

const PROGRAMS = [
  ['Google AI Studio', 'Google', 'Gemini free tier with RPM/RPD limits; Google account required.', 'https://aistudio.google.com'],
  ['OpenRouter Free', 'OpenRouter', 'Models with a :free suffix and rate limits; useful for compatibility tests.', 'https://openrouter.ai/models?max_price=0'],
  ['Groq Console', 'Groq', 'Developer free tier with per-model limits; low latency.', 'https://console.groq.com'],
  ['DeepSeek', 'DeepSeek', 'Trial credits for new accounts; changes with official campaigns.', 'https://platform.deepseek.com'],
  ['Alibaba Model Studio', 'Alibaba', 'Daily free quota for Qwen models; real-name verification required.', 'https://bailian.console.aliyun.com'],
  ['Moonshot', 'Moonshot', 'Trial credits for new accounts; Kimi models.', 'https://platform.moonshot.cn'],
];

export function EnFreePage() {
  const [models, setModels] = useState<ModelSummary[] | null>(null);

  useEffect(() => {
    document.title = 'Free tiers and trial credits for LLM APIs | RelayRadar';
    api.models().then((r) => setModels(r.items.filter((m) => m.freeTier))).catch(() => setModels([]));
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />FREE TIER</span>
          <h1>Free tiers and trial credits</h1>
          <p className="sub">Official free tiers and trial credits only; we do not list shared or resold keys. Quotas change often — confirm on the official console. <Link to="/free">中文免费额度</Link></p>
        </div>
      </div>

      {!models && <div className="loading"><Spinner /> Loading…</div>}

      <div className="free-grid">
        {models?.map((m) => (
          <article key={m.id} className="free-card panel">
            <div className="free-head">
              <div>
                <Link to={`/en/models/${m.id}`} className="free-name">{m.name}</Link>
                <span className="mono dim tiny">{m.id}</span>
              </div>
              <span className="free-tag">free tier</span>
            </div>
            <p className="free-note">{FREE_NOTE_EN[m.id] ?? m.summaryEn}</p>
            <dl className="free-kv">
              <div><dt>Official input</dt><dd className="mono">${m.inputUsd}</dd></div>
              <div><dt>Official output</dt><dd className="mono">${m.outputUsd}</dd></div>
              <div><dt>Cheapest input</dt><dd className="mono accent">{fmtCny(m.minInputCny)}</dd></div>
              <div><dt>Relays</dt><dd className="mono">{m.offersCount}</dd></div>
            </dl>
            <div className="free-foot">
              <span className="fine">Quotas and limits are set by the official console</span>
              <Link className="btn btn-ghost btn-sm" to={`/en/models/${m.id}`}>Prices →</Link>
            </div>
          </article>
        ))}
      </div>

      <section className="panel pad tips-panel">
        <h2>Official free programmes</h2>
        <p className="fine" style={{ marginBottom: 10 }}>Official entry points only; no shared keys. Quotas change — check the official page.</p>
        <div className="free-programs">
          {PROGRAMS.map(([name, vendor, note, url]) => (
            <div className="free-program" key={name}>
              <b>{name}</b>
              <span className="mono dim tiny">{vendor}</span>
              <p>{note}</p>
              <a className="link-btn" href={url} target="_blank" rel="noopener noreferrer nofollow">Official ↗</a>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}
