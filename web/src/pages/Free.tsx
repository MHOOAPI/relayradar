import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary } from '../types';
import { fmtCny, fmtUsd } from '../components/ui';
import { VendorMark } from '../components/marks';

const CONSOLE: Record<string, string> = {
  anthropic: 'https://console.anthropic.com',
  openai: 'https://platform.openai.com',
  google: 'https://aistudio.google.com',
  deepseek: 'https://platform.deepseek.com',
  qwen: 'https://bailian.console.aliyun.com',
  moonshot: 'https://platform.moonshot.cn',
  xai: 'https://console.x.ai',
};

const PROGRAMS = [
  { name: 'Google AI Studio', vendor: 'Google', note: 'Gemini 免费层，按 RPM/RPD 限速；需 Google 账号。', url: 'https://aistudio.google.com' },
  { name: 'OpenRouter Free', vendor: 'OpenRouter', note: '带 :free 后缀的模型，有速率限制；适合做兼容性测试。', url: 'https://openrouter.ai/models?max_price=0' },
  { name: 'Groq Console', vendor: 'Groq', note: '开发者免费层，按模型限速；延迟低，适合原型。', url: 'https://console.groq.com' },
  { name: 'DeepSeek 开放平台', vendor: 'DeepSeek', note: '新账号赠送额度，按官方活动变化。', url: 'https://platform.deepseek.com' },
  { name: '阿里云百炼', vendor: '阿里通义', note: 'Qwen 系列每日免费额度，需实名与开通。', url: 'https://bailian.console.aliyun.com' },
  { name: 'Moonshot 开放平台', vendor: 'Moonshot', note: '新账号试用额度，Kimi 系列。', url: 'https://platform.moonshot.cn' },
];

export function FreePage() {
  const [models, setModels] = useState<ModelSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.models()
      .then((r) => setModels(r.items.filter((m) => m.freeTier)))
      .catch((e: Error) => setError(e.message));
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>免费额度与试用</h1>
          <p className="sub">
            两条路：官方免费层与试用额度（自己注册、绑定结算账号），或社区共享 Key（随时可能失效）。
            这里只收录官方入口，不放来路不明的 Key。
          </p>
        </div>
        <span className="mono dim">
          {models ? `${models.length} 个目录内免费模型` : '读取中…'} · {PROGRAMS.length} 个官方计划
        </span>
      </div>

      <div className="tabs">
        <span className="tab on">官方免费计划</span>
        <Link className="tab" to="/free-api-keys">社区共享 Key</Link>
      </div>

      {error && <div className="error-box">{error}</div>}

      {models && models.length > 0 ? (
        <div className="free-grid">
          {models.map((m) => (
            <article key={m.id} className="free-card panel">
              <div className="free-head">
                <VendorMark vendor={m.vendor} size={34} />
                <div>
                  <Link to={`/models/${encodeURIComponent(m.id)}`} className="free-name">{m.name}</Link>
                  <span className="mono dim tiny">{m.id}</span>
                </div>
                <span className="free-tag">免费额度</span>
              </div>
              <p className="free-note">{m.freeNote}</p>
              <dl className="free-kv">
                <div><dt>官方输入</dt><dd className="mono">{fmtUsd(m.inputUsd)}</dd></div>
                <div><dt>官方输出</dt><dd className="mono">{fmtUsd(m.outputUsd)}</dd></div>
                <div><dt>中转最低</dt><dd className="mono accent">{fmtCny(m.minInputCny)}</dd></div>
                <div><dt>报价站点</dt><dd className="mono">{m.offersCount}</dd></div>
              </dl>
              <div className="free-foot">
                <span className="fine">额度与限速以官方控制台为准</span>
                <span className="free-actions">
                  {CONSOLE[m.vendor] && (
                    <a className="btn btn-ghost btn-sm" href={CONSOLE[m.vendor]} target="_blank" rel="noopener noreferrer nofollow">
                      官方控制台 ↗
                    </a>
                  )}
                  <Link className="btn btn-ghost btn-sm" to={`/models/${encodeURIComponent(m.id)}`}>看报价 →</Link>
                </span>
              </div>
            </article>
          ))}
        </div>
      ) : models && (
        <section className="panel free-empty">
          <b>目录里还没有标记免费层的模型</b>
          <p className="sub">
            免费层通常按月调整，这里只保留官方入口；想先试用，可以看下面的官方计划，或领一个社区共享 Key。
          </p>
          <div className="free-actions">
            <Link className="btn btn-primary btn-sm" to="/free-api-keys">看社区共享 Key</Link>
            <Link className="btn btn-ghost btn-sm" to="/models">浏览全部模型</Link>
          </div>
        </section>
      )}

      <section className="panel pad tips-panel">
        <h2>官方免费计划</h2>
        <p className="fine" style={{ marginBottom: 10 }}>只收录官方入口，不提供共享 Key；额度与限速以官方页面为准。</p>
        <div className="free-programs">
          {PROGRAMS.map((p) => (
            <div className="free-program" key={p.name}>
              <b>{p.name}</b>
              <span className="mono dim tiny">{p.vendor}</span>
              <p>{p.note}</p>
              <a className="link-btn" href={p.url} target="_blank" rel="noopener noreferrer nofollow">官方入口 ↗</a>
            </div>
          ))}
        </div>
      </section>

      <section className="panel pad tips-panel">
        <h2>免费额度使用建议</h2>
        <ul className="limits">
          <li>免费层通常有 RPM / RPD 限制，适合原型验证，不适合生产流量。</li>
          <li>绑定结算账号的“免费层”仍可能产生超额费用，先在控制台设置预算告警。</li>
          <li>赠送额度多与账号绑定，不要购买他人转卖的“免费 Key”，随时可能被回收。</li>
          <li>中转站的“免费测试金”不等于免费额度，通常要求充值后才可用。</li>
        </ul>
      </section>
    </main>
  );
}
