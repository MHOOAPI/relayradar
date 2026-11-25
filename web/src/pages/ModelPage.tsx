import '../styles/model.css';
import { Fragment, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelProviderStatus, ModelSummary, Offer, PriceChange, PriceDailyPoint } from '../types';
import { TrendChart } from '../components/TrendChart';
import { fmtCny, fmtDate, fmtUsd, Spinner, TierBadge } from '../components/ui';
import { StationMark, VendorMark } from '../components/marks';
import { PriceTexture } from '../components/Backdrop';
import { SectionNav } from '../components/SectionNav';

const MODEL_SECTIONS = [
  { id: 'score', label: '评分' },
  { id: 'price', label: '价格' },
  { id: 'trend', label: '走势' },
  { id: 'offers', label: '报价' },
  { id: 'changes', label: '变动' },
];

const fmtCtx = (n: number) => {
  if (!n || n <= 0) return '—';
  if (n >= 1000000) return `${Number((n / 1000000).toFixed(2))}M`;
  return `${Math.round(n / 1000)}K`;
};

const categoryLabel = (c?: string) => (c === 'image' ? '图片生成' : c === 'video' ? '视频生成' : '文本');
const unitLabel = (c?: string) => (c === 'image' ? '张' : c === 'video' ? '秒' : '百万 Token');
const scoreText = (v: number) => (v > 0 ? v.toFixed(1) : '—');

function ScoreBar({ label, value, note }: { label: string; value: number; note: string }) {
  return (
    <div className="score-row">
      <div className="score-head">
        <b>{label}</b>
        <span className="mono">{scoreText(value)}</span>
      </div>
      <div className="score-bar" aria-hidden="true">
        <i style={{ width: `${Math.max(0, Math.min(100, value))}%` }} />
      </div>
      <p className="score-note">{note}</p>
    </div>
  );
}

export function ModelPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [model, setModel] = useState<ModelSummary | null>(null);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [priceChanges, setPriceChanges] = useState<PriceChange[]>([]);
  const changeDays = useMemo(() => {
    const groups = new Map<string, PriceChange[]>();
    for (const c of priceChanges) {
      const d = new Date(c.at);
      const key = Number.isNaN(d.getTime()) ? '未知日期' : d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
      const list = groups.get(key);
      if (list) list.push(c);
      else groups.set(key, [c]);
    }
    return [...groups.entries()];
  }, [priceChanges]);
  const [allModels, setAllModels] = useState<ModelSummary[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [upstream, setUpstream] = useState<ModelProviderStatus | null>(null);
  const [offerSort, setOfferSort] = useState<'price' | 'availability' | 'rank' | 'multiplier'>('price');
  const [onlyOfficial, setOnlyOfficial] = useState(false);
  const [offerQ, setOfferQ] = useState('');
  const [relatedSort, setRelatedSort] = useState<'default' | 'price'>('default');
  const initialOfferLimit = () => (typeof window !== 'undefined' && window.matchMedia('(max-width: 640px)').matches ? 20 : 60);
  const [offerLimit, setOfferLimit] = useState(initialOfferLimit);
  const [priceHistory, setPriceHistory] = useState<PriceDailyPoint[]>([]);

  useEffect(() => {
    setOfferLimit(initialOfferLimit());
  }, [offerQ, onlyOfficial, offerSort]);

  useEffect(() => {
    if (!model) return;
    const map: Record<string, string> = {
      anthropic: 'anthropic', openai: 'openai', google: 'google', deepseek: 'deepseek',
      moonshot: 'moonshot', kimi: 'moonshot', qwen: 'alibaba', xai: 'xai',
    };
    let alive = true;
    api.modelStatus()
      .then((r) => {
        if (alive) setUpstream(r.providers.find((p) => p.key === map[model.vendor]) ?? null);
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [model?.vendor]);

  useEffect(() => {
    if (!id) return;
    setModel(null);
    api.model(id)
      .then((r) => {
        setModel(r.model);
        setOffers(r.offers ?? []);
        setPriceChanges(r.priceChanges ?? []);
      })
      .catch((e: Error) => setError(e.message));
  }, [id]);

  useEffect(() => {
    api.models().then((r) => setAllModels(r.items)).catch(() => setAllModels([]));
  }, []);

  useEffect(() => {
    if (!id) return;
    setPriceHistory([]);
    api.modelPriceHistory(id).then((r) => setPriceHistory(r.points ?? [])).catch(() => setPriceHistory([]));
  }, [id]);

  const related = useMemo(() => {
    if (!model) return [];
    const list = allModels.filter((m) => m.vendor === model.vendor && m.id !== model.id);
    if (relatedSort === 'price') {
      list.sort((a, b) => (a.minInputCny < 0 ? 1 : 0) - (b.minInputCny < 0 ? 1 : 0) || a.minInputCny - b.minInputCny);
    }
    return list.slice(0, 6);
  }, [allModels, model?.id, model?.vendor, relatedSort]);

  if (error) return <main className="wrap"><div className="error-box">{error}</div></main>;
  if (!model) return <main className="wrap"><div className="loading-page"><Spinner /> 读取模型…</div></main>;

  const best = offers.find((o) => o.purityTier !== 'counterfeit' && o.inputCny > 0);
  const officialCount = offers.filter((o) => o.purityTier === 'official').length;
  const offerNeedle = offerQ.trim().toLowerCase();
  const visibleOffers = offers
    .filter((o) => !onlyOfficial || o.purityTier === 'official')
    .filter((o) => !offerNeedle || `${o.stationName} ${o.stationSlug} ${o.groupName}`.toLowerCase().includes(offerNeedle))
    .sort((a, b) => {
      if (offerSort === 'availability') return b.availability - a.availability;
      if (offerSort === 'rank') return (a.stationRank || 1e9) - (b.stationRank || 1e9);
      if (offerSort === 'multiplier') return a.multiplier - b.multiplier;
      return a.inputCny - b.inputCny;
    });
  const shownOffers = visibleOffers.slice(0, offerLimit);
  const isText = !model.category || model.category === 'text';
  const perUnit = model.category === 'image' ? '/张' : model.category === 'video' ? '/秒' : '';
  const purityTotal = model.purityOfficial + model.purityMixed + model.purityUnknown + model.purityCounterfeit;

  const zhSummary = `${model.vendorName} 的${categoryLabel(model.category)}模型${isText ? `，上下文 ${fmtCtx(model.contextWindow)}` : ''}；官方价 ${model.inputUsd > 0 ? `${fmtUsd(model.inputUsd)} / ${fmtUsd(model.outputUsd)}` : '—'}${isText ? ' 每百万 Token' : ''}，${model.offersCount} 个站点有报价。`;

  const faq = [
    {
      q: `${model.name} 的官方价是多少？`,
      a: isText
        ? `官方标价 ${model.inputUsd > 0 ? fmtUsd(model.inputUsd) : '—'} 输入 / ${model.outputUsd > 0 ? fmtUsd(model.outputUsd) : '—'} 输出（每百万 Token）。中转站实价 = 官方美元价 × 站点汇率 × 分组倍率，所以同一模型在不同站点会差出数倍。`
        : `${model.name} 属于${categoryLabel(model.category)}，按${unitLabel(model.category)}计费；中转站会在此基础上叠加倍率，表中的最低价已折算成人民币。`,
    },
    {
      q: `${model.name} 全网最低价是多少，哪家站点？`,
      a: model.priceLowCny > 0
        ? `目前收录 ${model.offersCount} 个报价，最低 ${fmtCny(model.priceLowCny)}（${model.bestStation}），区间 ${fmtCny(model.priceLowCny)} – ${fmtCny(model.priceHighCny)}，中位 ${fmtCny(model.priceMedianCny)}。最低价不代表最稳，要同时看该站点的可用性与纯度。`
        : `目前没有收录到该模型的中转站报价，页面只展示官方价；一旦有站点上架并被同步，这里会自动出现实价。`,
    },
    {
      q: `怎么判断中转站的 ${model.name} 有没有被替换或降智？`,
      a: '用本站的接口检测对该站点发两次相同请求，核对模型回显、usage、分词一致性、缓存透传与首字延迟；这些都是可观测信号，检测不会直接对“降智”下结论。',
    },
    {
      q: `${model.name} 支持多长上下文？`,
      a: isText
        ? `目录记录为 ${fmtCtx(model.contextWindow)}。实际可用长度取决于站点是否透传长上下文，建议用长文档请求或接口检测验证。`
        : `${model.name} 是${categoryLabel(model.category)}模型，不按上下文窗口计费，主要看分辨率、时长与并发限制。`,
    },
    {
      q: `官方怎么介绍 ${model.name}？`,
      a: model.summary ? `${model.summary}（英文原文）` : '目录没有收录官方简介。',
    },
  ];

  return (
    <main className="wrap">
      <div className="crumbs mono">
        <Link to="/models">模型</Link><span>/</span><span className="dim">{model.id}</span>
      </div>

      {upstream && upstream.status !== 'operational' && (
        <div className={`upstream-banner ${upstream.status}`}>
          <b>{upstream.name} 上游状态：{upstream.status === 'degraded' ? '降级' : upstream.status === 'outage' ? '不可用' : '未知'}</b>
          <span>{upstream.description || '官方状态页暂未提供说明'}</span>
          <Link to={`/model-status/${upstream.key}`}>查看官方状态历史 →</Link>
        </div>
      )}

      <section className="panel detail-head model-hero has-wave">
        <PriceTexture
          offers={offers.map((o) => ({ price: o.inputCny, multiplier: o.multiplier, station: o.stationName, slug: o.stationSlug, tier: o.purityTier, rank: o.stationRank }))}
          onPick={(slug) => navigate(`/stations/${slug}`)}
        />
        <div className="detail-main">
          <VendorMark vendor={model.vendor} size={54} />
          <div>
            <div className="detail-title">
              <h1>{model.name}</h1>
              <span className="svc">{model.vendorName}</span>
            </div>
            <p className="tagline">{zhSummary}</p>
            <div className="detail-links mono dim">
              <span>{model.id}</span>
              <span>·</span>
              <span>{categoryLabel(model.category)}</span>
              {isText && <><span>·</span><span>上下文 {fmtCtx(model.contextWindow)}</span></>}
              <span>·</span>
              <Link to={`/models/license/${model.license}`}>许可证 {model.license}</Link>
              <span>·</span>
              <Link to={`/compare/models?a=${model.id}`}>对比模型</Link>
            </div>
          </div>
        </div>
        <div className="model-scores">
          <div><span>供给评分</span><b className="mono accent">{scoreText(model.relayScore)}</b></div>
          <div><span>最低价</span><b className="mono">{fmtCny(model.minInputCny)}{perUnit}</b></div>
          <div><span>报价站点</span><b className="mono">{model.offersCount}</b></div>
          {isText && <div><span>官方输入</span><b className="mono">{model.inputUsd > 0 ? fmtUsd(model.inputUsd) : '—'}</b></div>}
        </div>
      </section>

      <SectionNav items={MODEL_SECTIONS} />

      <div className="model-grid">
        <section className="panel" id="score">
          <div className="panel-head">
            <div>
              <h2>模型评分</h2>
              <p className="sub">按可观测的供给数据计算：价格、覆盖、可用性、纯度、上下文。不评价模型能力。</p>
            </div>
            <span className="mono dim score-total">供给评分 {scoreText(model.relayScore)} / 100</span>
          </div>
          <div className="score-grid">
            <ScoreBar label="价格" value={model.priceScore} note={
              model.priceLowCny > 0 && model.priceMedianCny > 0
                ? `最低 ${fmtCny(model.priceLowCny)}，中位 ${fmtCny(model.priceMedianCny)}（${model.priceRatio.toFixed(2)}×）`
                : '缺少报价，暂不计分'
            } />
            <ScoreBar label="覆盖" value={model.coverageScore} note={`${model.offersCount} 个站点有报价`} />
            <ScoreBar label="可用性" value={model.availabilityScore} note={`报价站点平均可用性 ${model.avgAvailability.toFixed(1)}`} />
            <ScoreBar label="纯度" value={model.purityScore} note={
              purityTotal > 0
                ? `官方直连 ${model.purityOfficial} · 混合 ${model.purityMixed} · 未检测 ${model.purityUnknown} · 疑似掺假 ${model.purityCounterfeit}`
                : '暂无纯度抽检'
            } />
            <ScoreBar label="上下文" value={model.contextScore} note={isText ? `窗口 ${fmtCtx(model.contextWindow)}` : '图片 / 视频模型按张、秒计费'} />
          </div>
          <p className="fine model-score-note">
            公式：价格 0.35 · 覆盖 0.20 · 可用性 0.20 · 纯度 0.15 · 上下文 0.10，按可用维度归一；价格分 = 100 ×（1 − 最低价 ÷ 中位价 ÷ 2）。
            <Link to="/methodology">评分方法 →</Link>
          </p>
        </section>

        <section className="panel" id="price">
          <div className="panel-head">
            <div>
              <h2>价格与供给</h2>
              <p className="sub">官方价来自厂商目录；实价来自站点公开倍率，按 7.2 CNY/USD 折算。</p>
            </div>
          </div>
          <dl className="fact-grid">
            {isText ? (
              <>
                <div><dt>官方输入 / 输出</dt><dd className="mono">{model.inputUsd > 0 ? `${fmtUsd(model.inputUsd)} / ${fmtUsd(model.outputUsd)}` : '—'}</dd></div>
                <div><dt>实价区间</dt><dd className="mono">{model.priceLowCny > 0 ? `${fmtCny(model.priceLowCny)} – ${fmtCny(model.priceHighCny)}` : '—'}</dd></div>
                <div><dt>实价中位</dt><dd className="mono">{model.priceMedianCny > 0 ? fmtCny(model.priceMedianCny) : '—'}</dd></div>
                <div><dt>最低 ÷ 中位</dt><dd className={`mono ${model.priceRatio > 0 && model.priceRatio < 1 ? 'delta-down' : ''}`}>{model.priceLowCny > 0 && model.priceMedianCny > 0 ? `${model.priceRatio.toFixed(2)}×` : '—'}</dd></div>
                <div><dt>上下文</dt><dd className="mono">{fmtCtx(model.contextWindow)}</dd></div>
                <div><dt>发布日期</dt><dd className="mono">{model.releasedAt && !model.releasedAt.startsWith('0001') ? fmtDate(model.releasedAt) : '—'}</dd></div>
              </>
            ) : (
              <>
                <div><dt>计费单位</dt><dd className="mono">按{unitLabel(model.category)}</dd></div>
                <div><dt>最低价</dt><dd className="mono">{fmtCny(model.minInputCny)}{perUnit}</dd></div>
                <div><dt>报价站点</dt><dd className="mono">{model.offersCount}</dd></div>
                <div><dt>平均可用性</dt><dd className="mono">{model.avgAvailability.toFixed(1)}</dd></div>
              </>
            )}
          </dl>
          <div className="supply-foot">
            <span className="mono dim">最低价站点</span>
            <b>{model.bestStation || '暂无报价'}</b>
            <Link className="link-btn mono" to="/audit">检测接口 →</Link>
          </div>
        </section>
      </div>

      <section className="panel price-trend" id="trend">
        <div className="panel-head">
          <div>
            <h2>全网最低价走势</h2>
            <p className="sub">每天记录一次各站最低输入价；系统从部署当天开始积累，曲线随天数增长。</p>
          </div>
          <div className="panel-head-right">
            <span className="mono dim">
              {priceHistory.length > 0
                ? `${priceHistory.length} 天 · 最新 ${fmtCny(priceHistory[priceHistory.length - 1].minInputCny)}`
                : '尚未记录'}
            </span>
            <Link className="btn btn-ghost btn-xs" to={`/changes?subscribe=1&model=${encodeURIComponent(model.id)}`}>
              订阅该模型降价
            </Link>
          </div>
        </div>
        {priceHistory.length >= 2 ? (
          <div className="price-trend-chart">
            <TrendChart
              values={priceHistory.map((p) => p.minInputCny)}
              labels={priceHistory.map((p) => p.day.slice(5))}
              color="#2ee6a6"
              unit=" 元"
              decimals={2}
              height={160}
            />
          </div>
        ) : (
          <div className="price-trend-empty">
            <b>
              {priceHistory.length === 1
                ? `今天全网最低 ${fmtCny(priceHistory[0].minInputCny)}`
                : '还没有价格快照'}
            </b>
            <p className="sub">
              {priceHistory.length === 1
                ? `已记录 ${priceHistory[0].stations} 个站点的报价，累计 2 天后这里显示走势曲线。`
                : '快照任务每天运行一次，部署后的第二天开始有数据。'}
            </p>
            <a className="link-btn mono" href="#offers">先看当前 {offers.length} 条报价 →</a>
          </div>
        )}
      </section>

      <section className="panel model-offers" id="offers">
        <div className="panel-head">
          <div>
            <h2>全网报价</h2>
            <p className="sub">同一模型可能在不同分组的倍率下差价数倍，注意同时看纯度与可用性。</p>
          </div>
          <span className="mono dim" aria-live="polite">
            {visibleOffers.length === offers.length ? `${offers.length} 条报价` : `${visibleOffers.length} / ${offers.length} 条报价`}
            {best && ` · 最低 ${fmtCny(best.inputCny)} · ${best.stationName}`}
          </span>
        </div>
        <details className="price-basis">
          <summary>价格口径与换算</summary>
          <ul>
            <li>官方价取自厂商公开定价页，单位为美元 / 百万 Token；没有公开价的模型显示 —。</li>
            <li>本站人民币价 = 美元价 × 7.2 汇率 × 分组倍率；倍率来自站点公开的分组设置。</li>
            <li>缓存命中、批处理、阶梯价与图片/视频计费可能另算，表中只反映标准输入与输出价。</li>
            <li>价格快照每天记录一次，页面显示的是最近一次观测值，不等同于站点实时结算价。</li>
          </ul>
        </details>
        <div className="offer-tools">
          <div className="chips">
            <button
              type="button"
              className={`chip ${!onlyOfficial ? 'on' : ''}`}
              aria-pressed={!onlyOfficial}
              onClick={() => setOnlyOfficial(false)}
            >
              全部报价 <span className="chip-count">{offers.length}</span>
            </button>
            <button
              type="button"
              className={`chip ${onlyOfficial ? 'on' : ''}`}
              aria-pressed={onlyOfficial}
              onClick={() => setOnlyOfficial(true)}
            >
              官方直连 <span className="chip-count">{officialCount}</span>
            </button>
          </div>
          <div className="offer-search">
            <input
              value={offerQ}
              onChange={(e) => setOfferQ(e.target.value)}
              placeholder="筛选站点 / 分组"
              aria-label="筛选报价"
            />
          </div>
          <label className="chip-select">
            <span className="dim">排序</span>
            <select value={offerSort} onChange={(e) => setOfferSort(e.target.value as typeof offerSort)} aria-label="报价排序">
              <option value="price">输入价从低到高</option>
              <option value="availability">可用性从高到低</option>
              <option value="rank">站点排名从高到低</option>
              <option value="multiplier">倍率从低到高</option>
            </select>
          </label>
        </div>
        <div className="table-wrap offer-table">
          <table className="index-table">
            <thead>
              <tr>
                <th className="num">#</th><th>站点</th><th>分组</th><th className="num">倍率</th>
                <th className="num">{perUnit ? `价格 ${perUnit}` : '输入 /1M'}</th><th className="num">{perUnit ? '同价' : '输出 /1M'}</th><th>纯度</th><th className="num">可用性指数</th>
              </tr>
            </thead>
            <tbody>
              {visibleOffers.length === 0 && (
                <tr><td colSpan={8} className="loading">{offers.length === 0 ? '暂无中转站报价；以上为官方价参考。' : '没有符合筛选的报价，试试看全部。'}</td></tr>
              )}
              {shownOffers.map((o, i) => (
                <tr key={`${o.stationSlug}-${o.groupSlug}`} className={o === best ? 'offer-best' : undefined}>
                  <td className="num rank dim">{i + 1}</td>
                  <td>
                    <div className="cell-station">
                      <StationMark slug={o.stationSlug} name={o.stationName} iconUrl={o.iconUrl} size={26} />
                      <Link className="station-name" to={`/stations/${o.stationSlug}`}>{o.stationName}</Link>
                      {o.claimed && <span className="badge-ok" title="站长已认领">✓</span>}
                      {o === best && <span className="best-chip">最低价</span>}
                    </div>
                  </td>
                  <td><span className="mono dim">{o.groupName}</span></td>
                  <td className="num mono">{o.multiplier.toFixed(2)}×</td>
                  <td className="num mono">{fmtCny(o.inputCny)}{perUnit}</td>
                  <td className="num mono accent">{fmtCny(o.outputCny)}{perUnit}</td>
                  <td><TierBadge tier={o.purityTier} /></td>
                  <td className="num mono dim">{o.availability > 0 ? o.availability.toFixed(1) : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="offer-cards">
          {shownOffers.map((o, i) => (
            <Link key={`${o.stationSlug}-${o.groupSlug}`} to={`/stations/${o.stationSlug}`} className={`offer-card${o === best ? ' best' : ''}`}>
              <div className="oc-head">
                <span className="mono dim">{i + 1}</span>
                <StationMark slug={o.stationSlug} name={o.stationName} iconUrl={o.iconUrl} size={28} />
                <div className="oc-id">
                  <b>{o.stationName}</b>
                  <small className="mono">{o.groupName}</small>
                </div>
                {o.claimed && <span className="badge-ok" title="站长已认领">✓</span>}
              </div>
              {o === best && <span className="best-chip">最低价</span>}
              <div className="oc-prices">
                <span>输入 <b className="mono">{fmtCny(o.inputCny)}</b></span>
                <span>输出 <b className="mono">{fmtCny(o.outputCny)}</b></span>
                <span>倍率 <b className="mono">{o.multiplier.toFixed(2)}×</b></span>
              </div>
              <div className="oc-foot">
                <TierBadge tier={o.purityTier} />
                <span className="mono dim">可用性 {o.availability > 0 ? o.availability.toFixed(1) : '—'}</span>
              </div>
            </Link>
          ))}
        </div>
        {visibleOffers.length > shownOffers.length && (
          <div className="list-more">
            <button className="btn btn-ghost btn-sm" onClick={() => setOfferLimit((v) => v + 60)}>
              显示更多（剩余 {visibleOffers.length - shownOffers.length}）
            </button>
            <span className="mono">已显示 {shownOffers.length} / {visibleOffers.length}</span>
          </div>
        )}
      </section>

      {priceChanges.length > 0 && (
        <section className="panel" id="changes">
          <div className="panel-head">
            <div>
              <h2>价格变动记录</h2>
              <p className="sub">来自历史价格快照，只记录实际观测到的调整；未列出的站点代表价格未变。</p>
            </div>
            <span className="mono dim">{priceChanges.length} 次</span>
          </div>
          <div className="table-wrap">
            <table className="index-table">
              <thead>
                <tr>
                  <th>站点</th><th>分组</th><th className="num">输入价</th>
                  <th className="num">变动</th><th>记录时间</th>
                </tr>
              </thead>
              <tbody>
                {changeDays.map(([day, list]) => (
                  <Fragment key={day}>
                    <tr className="day-row">
                      <td colSpan={5} className="mono dim">{day} · {list.length} 次</td>
                    </tr>
                    {list.map((c) => (
                      <tr key={`${c.stationSlug}-${c.groupName}-${c.at}`}>
                        <td><Link className="station-name" to={`/stations/${c.stationSlug}`}>{c.stationName}</Link></td>
                        <td><span className="mono dim">{c.groupName}</span></td>
                        <td className="num mono">{fmtCny(c.oldInputCny)} → {fmtCny(c.newInputCny)}</td>
                        <td className={`num mono ${c.deltaPct > 0 ? 'delta-up' : 'delta-down'}`}>
                          {c.deltaPct > 0 ? '+' : ''}{c.deltaPct.toFixed(1)}%
                        </td>
                        <td><span className="mono dim">{fmtDate(c.at)}</span></td>
                      </tr>
                    ))}
                  </Fragment>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {related.length > 0 && (
        <section className="panel">
          <div className="panel-head">
            <div>
              <h2>同厂商模型</h2>
              <p className="sub">同一厂商的其他型号，价格与上下文可横向对比。</p>
            </div>
            <select
              className="related-sort"
              value={relatedSort}
              onChange={(e) => setRelatedSort(e.target.value as typeof relatedSort)}
              aria-label="同厂商模型排序"
            >
              <option value="default">默认顺序</option>
              <option value="price">价格从低到高</option>
            </select>
            <Link className="link-btn mono" to={`/models?vendor=${encodeURIComponent(model.vendor)}`}>全部 {model.vendorName} 模型 →</Link>
          </div>
          <div className="related-grid">
            {related.map((m) => (
              <Link key={m.id} className="related-card" to={`/models/${encodeURIComponent(m.id)}`}>
                <div className="rc-head">
                  <VendorMark vendor={m.vendor} size={22} />
                  <b>{m.name}</b>
                </div>
                <div className="rc-meta mono dim">
                  {m.contextWindow > 0 ? `${fmtCtx(m.contextWindow)} 上下文` : categoryLabel(m.category)}
                  {m.offersCount > 0 ? ` · ${m.offersCount} 站报价` : ' · 暂无报价'}
                </div>
                <div className="rc-price mono">{m.minInputCny >= 0 ? fmtCny(m.minInputCny) : '—'}</div>
              </Link>
            ))}
          </div>
        </section>
      )}

      <section className="panel model-faq">
        <div className="panel-head"><h2>常见问题</h2></div>
        <div className="faq-list">
          {faq.map((f) => (
            <details key={f.q} className="faq-item">
              <summary>{f.q}</summary>
              <p>{f.a}</p>
            </details>
          ))}
        </div>
      </section>
    </main>
  );
}
