import { Link } from 'react-router-dom';

const PLACEMENTS = [
  { name: '首页赞助条', price: '¥149 / 月', desc: '首页顶部一条文字赞助，标注 Sponsor，不进入榜单。', slot: '1 个位置' },
  { name: '目录赞助卡', price: '¥249 / 月', desc: '站点目录侧栏卡片，展示 logo、简介与链接。', slot: '2 个位置' },
  { name: '榜单页赞助位', price: '¥399 / 月', desc: '排行榜页固定赞助位，明确标注且与自然结果分离。', slot: '1 个位置' },
];

export function SponsorPage() {
  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />SPONSOR</span>
          <h1>赞助与广告</h1>
          <p className="sub">赞助位用于覆盖探针与带宽成本。它不会改变分数、排序或测评结论。</p>
        </div>
      </div>

      <section className="panel pad sponsor-rules">
        <h2>我们遵守的规则</h2>
        <ul className="limits">
          <li>所有赞助内容标注 <b>Sponsor</b>，不与自然结果混排，不使用“推荐”伪装。</li>
          <li>赞助不参与评分与排序，也不能要求删除负面检测结果。</li>
          <li>文案与数据分离：赞助方提供介绍，评分与探针数据由本站独立生成。</li>
          <li>不接受资金盘、共享 Key、无法核实主体的投放。</li>
        </ul>
      </section>

      <div className="sponsor-grid">
        {PLACEMENTS.map((p) => (
          <article key={p.name} className="panel sponsor-card">
            <span className="sponsor-slot mono">{p.slot}</span>
            <h2>{p.name}</h2>
            <b className="sponsor-price mono">{p.price}</b>
            <p>{p.desc}</p>
            <span className="fine">示例报价，最终以合同为准</span>
          </article>
        ))}
      </div>

      <section className="panel pad">
        <h2>投放流程</h2>
        <ol className="limits">
          <li>邮件联系 <a href="mailto:sponsor@relayradar.example">sponsor@relayradar.example</a>，说明产品与目标页面。</li>
          <li>双方确认位置、周期与文案；赞助方提供 logo 与落地页。</li>
          <li>上线前标注 Sponsor；周期结束后自动下架，不自动续约。</li>
        </ol>
        <p className="fine">
          当前为演示部署，页面与报价仅用于展示流程，不构成真实要约。想先看数据口径，请读 <Link to="/methodology">评分方法</Link>。
        </p>
      </section>
    </main>
  );
}
