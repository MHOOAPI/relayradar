import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary } from '../types';
import { fmtCny, fmtUsd, Spinner } from '../components/ui';
import { VendorMark } from '../components/marks';

const ctx = (n: number) => (n >= 1000000 ? `${(n / 1000000).toFixed(n % 1000000 === 0 ? 0 : 2)}M` : `${Math.round(n / 1000)}K`);

export function ModelComparePage() {
  const [params, setParams] = useSearchParams();
  const [models, setModels] = useState<ModelSummary[] | null>(null);
  const [a, setA] = useState(params.get('a') ?? '');
  const [b, setB] = useState(params.get('b') ?? '');
  const [qA, setQA] = useState('');
  const [qB, setQB] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.models()
      .then((r) => {
        setModels(r.items);
        const first = r.items[0]?.id ?? '';
        const second = r.items[1]?.id ?? first;
        setA((cur) => cur || first);
        setB((cur) => cur || second);
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  useEffect(() => {
    const p = new URLSearchParams();
    if (a) p.set('a', a);
    if (b) p.set('b', b);
    setParams(p, { replace: true });
  }, [a, b, setParams]);

  const left = useMemo(() => models?.find((m) => m.id === a) ?? null, [models, a]);
  const right = useMemo(() => models?.find((m) => m.id === b) ?? null, [models, b]);

  const filter = (q: string) => {
    const needle = q.trim().toLowerCase();
    return (models ?? []).filter((m) => !needle || `${m.name} ${m.id} ${m.vendorName}`.toLowerCase().includes(needle));
  };
  const listA = filter(qA);
  const listB = filter(qB);

  if (error) return <main className="wrap"><div className="error-box">{error}</div></main>;
  if (!models) return <main className="wrap"><div className="loading-page"><Spinner /> 读取模型…</div></main>;

  const rows: Array<[string, (m: ModelSummary) => string]> = [
    ['厂商', (m) => m.vendorName],
    ['上下文', (m) => ctx(m.contextWindow)],
    ['官方输入 /1M', (m) => fmtUsd(m.inputUsd)],
    ['官方输出 /1M', (m) => fmtUsd(m.outputUsd)],
    ['全网最低输入', (m) => fmtCny(m.minInputCny)],
    ['编程分', (m) => m.codingScore.toFixed(1)],
    ['推理分', (m) => m.reasoningScore.toFixed(1)],
    ['报价站点', (m) => String(m.offersCount)],
    ['许可证', (m) => m.license],
  ];

  const better = (key: string, l: ModelSummary, r: ModelSummary) => {
    if (key === '全网最低输入') return l.minInputCny < r.minInputCny ? 'l' : r.minInputCny < l.minInputCny ? 'r' : '';
    if (key === '编程分') return l.codingScore > r.codingScore ? 'l' : r.codingScore > l.codingScore ? 'r' : '';
    if (key === '推理分') return l.reasoningScore > r.reasoningScore ? 'l' : r.reasoningScore > l.reasoningScore ? 'r' : '';
    return '';
  };

  const picker = (
    label: string,
    value: string,
    other: string,
    set: (id: string) => void,
    q: string,
    setQ: (q: string) => void,
    list: ModelSummary[],
  ) => (
    <div className="mp-col">
      <div className="mp-head">
        <span className="mp-label mono">{label}</span>
        <b>{models.find((m) => m.id === value)?.name ?? '未选择'}</b>
      </div>
      <input className="mp-search" value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索模型 / 厂商 / ID" aria-label={`${label} 搜索`} />
      <div className="mp-list">
        {[...list].sort((x, y) => (x.id === value ? 0 : 1) - (y.id === value ? 0 : 1)).map((m) => (
          <button
            key={m.id}
            type="button"
            className={`mp-item ${m.id === value ? 'on' : ''}`}
            disabled={m.id === other}
            onClick={() => set(m.id)}
          >
            <VendorMark vendor={m.vendor} size={18} />
            <span>{m.name}</span>
            <small>{m.vendorName}</small>
            {m.id === value && <i className="mp-check">✓</i>}
          </button>
        ))}
        {list.length === 0 && <p className="fine" style={{ padding: 10 }}>没有匹配的模型</p>}
      </div>
    </div>
  );

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/models">模型</Link><span>/</span><span className="dim">对比</span></div>
      <div className="page-head">
        <div>
          <h1>模型对比</h1>
          <p className="sub">同一张表里看官方价、全网最低实价与能力分；绿色为该行更优。选择结果写进 URL，可直接分享。</p>
        </div>
        <div className="head-actions">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={() => {
              setA(b);
              setB(a);
            }}
            disabled={!a || !b}
          >
            ⇄ 交换
          </button>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setA(''); setB(''); }}>
            清空
          </button>
        </div>
      </div>

      <section className="panel model-compare-pickers">
        {picker('模型 A', a, b, setA, qA, setQA, listA)}
        <div className="mp-vs mono">vs</div>
        {picker('模型 B', b, a, setB, qB, setQB, listB)}
      </section>

      {left && right && (
        <section className="panel">
          <p className="fine compare-hint">左右滑动查看全部对比，首列固定。</p>
          <div className="table-wrap compare-wrap">
            <table className="index-table compare-models">
              <thead>
                <tr>
                  <th>对比维度</th>
                  <th><span className="mc-head"><VendorMark vendor={left.vendor} size={22} />{left.name}</span></th>
                  <th><span className="mc-head"><VendorMark vendor={right.vendor} size={22} />{right.name}</span></th>
                </tr>
              </thead>
              <tbody>
                {rows.map(([label, fn]) => {
                  const win = better(label, left, right);
                  return (
                    <tr key={label}>
                      <td className="row-label">{label}</td>
                      <td className={win === 'l' ? 'win mono' : 'mono'}>{fn(left)}</td>
                      <td className={win === 'r' ? 'win mono' : 'mono'}>{fn(right)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </main>
  );
}
