import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { WindowRow } from '../types';
import { Spinner } from '../components/ui';

export function WindowLeaderboardPage() {
  const [days, setDays] = useState(7);
  const [rows, setRows] = useState<WindowRow[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setRows(null);
    api.windowLeaderboard(days)
      .then((r) => setRows(r.items))
      .catch((e: Error) => setError(e.message));
  }, [days]);

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/leaderboards">排行榜</Link><span>/</span><span className="dim">周榜 / 月榜</span></div>
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />WINDOW LEADERBOARD</span>
          <h1>周榜 / 月榜</h1>
          <p className="sub">按窗口内每日综合分取均值，和当前分对比，减少单日波动的影响。</p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/leaderboards">返回排行榜</Link>
      </div>

      {error && <div className="error-box">{error}</div>}

      <div className="tabs">
        <button className={`tab ${days === 7 ? 'on' : ''}`} onClick={() => setDays(7)}>近 7 天</button>
        <button className={`tab ${days === 30 ? 'on' : ''}`} onClick={() => setDays(30)}>近 30 天</button>
        <span className="tabs-note">历史不足 30 天时自动使用可用区间，Days 列会标注实际天数。</span>
      </div>

      <section className="panel">
        <div className="table-wrap">
          <table className="index-table">
            <thead>
              <tr>
                <th className="col-rank">#</th>
                <th>站点</th>
                <th className="num">窗口均分</th>
                <th className="num">当前分</th>
                <th className="num">差值</th>
                <th className="num">窗口可用率</th>
                <th className="num hide-sm">实际天数</th>
              </tr>
            </thead>
            <tbody>
              {!rows && <tr><td colSpan={8} className="loading"><Spinner /> 计算窗口…</td></tr>}
              {rows?.map((r, i) => (
                <tr key={r.slug}>
                  <td className={`col-rank rank ${i < 3 ? `r${i + 1}` : ''}`}>{i + 1}</td>
                  <td><Link className="station-name" to={`/stations/${r.slug}`}>{r.name}</Link></td>
                  <td className="num mono">{r.average.toFixed(1)}</td>
                  <td className="num mono dim">{r.current.toFixed(1)}</td>
                  <td className={`num mono ${r.delta >= 0 ? 'delta-pos' : 'delta-neg'}`}>
                    {r.delta >= 0 ? '+' : ''}{r.delta.toFixed(1)}
                  </td>
                  <td className="num mono">{r.uptime.toFixed(1)}%</td>
                  <td className="num mono dim hide-sm">{r.days}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="panel-foot mono dim">
          窗口均分 = 每日综合分的算术平均；差值 = 窗口均分 − 当前分。月度窗口需要至少 30 天历史。
        </div>
      </section>
    </main>
  );
}
