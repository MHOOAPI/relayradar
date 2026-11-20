import { useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api';
import type { StationDetail } from '../types';
import { PayMark } from '../components/pay';
import { StationMark } from '../components/marks';
import { Spinner } from '../components/ui';

export function GoPage() {
  const { slug = '' } = useParams();
  const [data, setData] = useState<StationDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [auto, setAuto] = useState(() => localStorage.getItem('rr-go-auto') !== '0');
  const [seconds, setSeconds] = useState(8);
  const [cancelled, setCancelled] = useState(false);
  const tracked = useRef(false);

  useEffect(() => {
    setSeconds(8);
    setCancelled(false);
    tracked.current = false;
    api.station(slug)
      .then((d) => {
        setData(d);
        document.title = `即将离开：${d.station.name} | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [slug]);

  useEffect(() => {
    if (!data || tracked.current) return;
    tracked.current = true;
    api.outbound(slug).catch(() => {});
  }, [data, slug]);

  useEffect(() => {
    if (!data || !auto || cancelled) return;
    if (seconds <= 0) {
      window.location.assign(`https://${data.station.domain}`);
      return;
    }
    const t = window.setTimeout(() => setSeconds((v) => v - 1), 1000);
    return () => window.clearTimeout(t);
  }, [data, auto, cancelled, seconds]);

  if (error) {
    return (
      <main className="wrap">
        <div className="error-box">
          {error} <Link className="link-btn" to="/">返回榜单</Link>
        </div>
      </main>
    );
  }
  if (!data) {
    return <main className="wrap"><div className="loading-page"><Spinner /> 读取站点信息…</div></main>;
  }

  const s = data.station;
  const target = `https://${s.domain}`;

  return (
    <main className="wrap">
      <section className="panel go-card">
        <span className="eyebrow"><i />LEAVING RELAYRADAR</span>
        <h1>即将离开本站，前往 {s.name}</h1>

        <div className="go-target">
          <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={44} />
          <div>
            <b>{s.name}</b>
            <span className="mono dim">{s.domain}</span>
          </div>
          <span className="go-arrow mono">→</span>
          <span className="mono go-url">{s.baseUrl}</span>
        </div>

        <p className="go-lead">
          RelayRadar 不参与交易、不收款、不担保售后。离开前请花 30 秒确认下面几件事：
        </p>

        <ul className="go-checklist">
          <li><b>核对域名</b>：目标站应为 <code>{s.domain}</code>，Base URL 与档案一致，不要在搜索广告里点错。</li>
          <li><b>先小额</b>：首次充值不超过 <b>¥{s.minTopupCny}</b>，使用可随时吊销的临时 Key。</li>
          <li><b>验证计费</b>：充值后先跑 1 美元额度，核对扣费是否等于「官方价 × 汇率 × 倍率」。</li>
          <li><b>保留凭证</b>：截图充值记录与退款政策，确认发票类型；不要向个人账户转账。</li>
        </ul>

        {s.payments.length > 0 && (
          <div className="go-pay">
            <span className="mono dim">对方支持的支付方式</span>
            <span className="pay-list">
              {s.payments.map((p) => (
                <span className="pay-chip" key={p}><PayMark method={p} size={15} />{p}</span>
              ))}
            </span>
          </div>
        )}

        <div className="go-actions">
          <a className="btn btn-primary" href={target} target="_blank" rel="noopener noreferrer nofollow">
            继续访问 {s.domain}
            {auto && !cancelled ? `（${seconds}s）` : ''} ↗
          </a>
          {auto && !cancelled && (
            <button type="button" className="btn btn-ghost" onClick={() => setCancelled(true)}>
              暂停自动跳转
            </button>
          )}
          <Link className="btn btn-ghost" to={`/stations/${s.slug}`}>返回档案</Link>
          <Link className="btn btn-ghost" to="/">回榜单</Link>
        </div>

        <label className="check-inline go-auto">
          <input
            type="checkbox"
            checked={!auto}
            onChange={(e) => {
              const skip = e.target.checked;
              setAuto(!skip);
              setSeconds(8);
              setCancelled(false);
              localStorage.setItem('rr-go-auto', skip ? '0' : '1');
            }}
          />
          以后不再自动跳转（仍会显示本页）
        </label>

        <p className="fine">
          本站不对目标站点的内容、可用性、资金安全与售后负责；链接会以 nofollow 标记打开。
        </p>
      </section>
    </main>
  );
}
