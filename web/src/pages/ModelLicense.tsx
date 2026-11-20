import { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary } from '../types';
import { fmtCny, fmtUsd, Spinner } from '../components/ui';
import { VendorMark } from '../components/marks';

const LABEL: Record<string, string> = {
  'proprietary': '闭源 / 专有',
  'mit': 'MIT',
  'apache-2.0': 'Apache-2.0',
  'modified-mit': 'Modified MIT',
};
const ORDER = ['proprietary', 'mit', 'apache-2.0', 'modified-mit'];

export function ModelLicensePage() {
  const { license = '' } = useParams();
  const [models, setModels] = useState<ModelSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.models()
      .then((r) => setModels(r.items))
      .catch((e: Error) => setError(e.message));
  }, []);

  const grouped = useMemo(() => {
    const map = new Map<string, ModelSummary[]>();
    for (const m of models ?? []) {
      const list = map.get(m.license) ?? [];
      list.push(m);
      map.set(m.license, list);
    }
    return map;
  }, [models]);

  if (error) return <main className="wrap"><div className="error-box">{error}</div></main>;
  if (!models) return <main className="wrap"><div className="loading-page"><Spinner /> 读取模型…</div></main>;

  if (!license) {
    return (
      <main className="wrap">
        <div className="page-head">
          <div>
            <h1>按许可证浏览模型</h1>
            <p className="sub">许可证决定权重能否自托管、能否商用；这里按许可证聚合模型与实价。</p>
          </div>
        </div>
        <div className="license-grid">
          {ORDER.filter((k) => grouped.has(k)).map((k) => (
            <Link key={k} to={`/models/license/${k}`} className="panel license-card">
              <b>{LABEL[k] ?? k}</b>
              <span className="mono dim">{grouped.get(k)?.length} 个模型</span>
              <small>{grouped.get(k)?.slice(0, 4).map((m) => m.name).join(' · ')}</small>
            </Link>
          ))}
        </div>
      </main>
    );
  }

  const list = grouped.get(license) ?? [];
  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/models">模型</Link><span>/</span><Link to="/models/license">许可证</Link><span>/</span><span className="dim">{license}</span></div>
      <div className="page-head">
        <div>
          <h1>{LABEL[license] ?? license} 许可证模型</h1>
          <p className="sub">{list.length} 个模型使用该许可证。</p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/models/license">全部许可证</Link>
      </div>
      <section className="panel">
        <div className="table-wrap">
          <table className="index-table">
            <thead><tr><th>模型</th><th>厂商</th><th className="num">官方价/1M</th><th className="num">全网最低输入</th></tr></thead>
            <tbody>
              {list.map((m) => (
                <tr key={m.id}>
                  <td>
                    <div className="cell-station">
                      <VendorMark vendor={m.vendor} size={24} />
                      <div><Link className="station-name" to={`/models/${m.id}`}>{m.name}</Link><span className="mono dim tiny">{m.id}</span></div>
                    </div>
                  </td>
                  <td>{m.vendorName}</td>
                  <td className="num mono">{fmtUsd(m.inputUsd)} / {fmtUsd(m.outputUsd)}</td>
                  <td className="num mono accent">{fmtCny(m.minInputCny)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </main>
  );
}
