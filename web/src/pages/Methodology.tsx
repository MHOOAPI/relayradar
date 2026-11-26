import { Link } from 'react-router-dom';

const REFS = [
  { name: 'OkkMax', url: 'https://www.okkmax.com', desc: '公开的分组评分方法与逐站分组数据' },
  { name: 'VeriDrop', url: 'https://veridrop.app', desc: '开源的思考签名交叉验证检测器' },
  { name: 'LMSpeed', url: 'https://lmspeed.net', desc: '模型跑分与多轮速度基准' },
  { name: 'PriceAI', url: 'https://priceai.cc', desc: '卡网订阅与中转 API 的比价口径' },
  { name: '禾维 AI', url: 'https://www.hvoyai.com', desc: '面向 Claude 中转的检测实践' },
];

export function MethodologyPage() {
  return (
    <main className="wrap method">
      <div className="page-head">
        <div>
          <h1>评分方法 v2</h1>
          <p className="sub">
            所有分数都能用手算复现：三个 0–100 的轴、两组公开权重、一个明确的样本门槛。
            付费、赞助与商务合作不改变其中任何一项。
          </p>
        </div>
        <span className="ver-pill mono">2026-11 修订</span>
      </div>

      <section className="panel pad">
        <h2>0. 为什么从 v1 改到 v2</h2>
        <p className="sub">v1 的四个问题在真实数据上会得出反直觉的结果，v2 逐一修掉：</p>
        <table className="tier-table versions">
          <thead><tr><th>v1 做法</th><th>问题</th><th>v2 做法</th></tr></thead>
          <tbody>
            <tr><td>70% 地板 + 平方映射</td><td>96% 原始可用率被压成 76 分，和页面展示的可用率对不上</td><td>Wilson 95% 置信下界，数值与样本量直接相关</td></tr>
            <tr><td>速度只作为可用性的乘数</td><td>速度 100 时，排名分恰好等于可用性分，看起来像 bug</td><td>三个轴独立计算后加权，永远可分解</td></tr>
            <tr><td>0.8ⁱ 衰减权重</td><td>权重来源无法解释，分组多的站点吃亏</td><td>80% 覆盖加权均值 + 20% 最差分组，规则公开</td></tr>
            <tr><td>12 条探针即可计分</td><td>少量幸运探针能刷出接近满分</td><td>每组至少 24 条、窗口 7 天，才进入计分</td></tr>
          </tbody>
        </table>
      </section>

      <section className="panel pad">
        <h2>1. 每个分组：三个轴</h2>
        <div className="method-grid">
          <div>
            <h3>可用性 A</h3>
            <p>
              取最近 7 天探针，成功 = <code>ok</code> 或 <code>slow</code>。用 Wilson 95% 置信区间的
              <strong>下界</strong>作为分数，而不是简单成功率：84/84 次成功得 95.6 分，100/100 得 96.3 分，
              样本越少越拿不到满分。
            </p>
            <pre className="formula-block mono">{`A = 100 × WilsonLB(k, n, z = 1.96)`}</pre>
          </div>
          <div>
            <h3>速度 S</h3>
            <p>
              只统计成功探针的延迟中位数 <code>p50</code>，线性映射到 0–100：800ms 及以内满分，
              5000ms 及以上归零。样本少于 3 条时取中性值 50。
            </p>
            <pre className="formula-block mono">{`S = 100 × clamp((5000 − p50) / 4200, 0, 1)`}</pre>
          </div>
          <div>
            <h3>稳定性 C</h3>
            <p>
              比较 <code>p95</code> 与 <code>p50</code> 的抖动比例：完全不抖是 100 分，
              p95 达到 p50 的 3 倍时归零。样本少于 10 条时取中性值 60。
            </p>
            <pre className="formula-block mono">{`jitter = (p95 − p50) / max(p50, 150ms)
C = 100 × clamp(1 − jitter / 2, 0, 1)`}</pre>
          </div>
          <div>
            <h3>分组质量</h3>
            <p>用于分组表展示，也是站点聚合的输入：</p>
            <pre className="formula-block mono">{`Q = 0.62·A + 0.26·S + 0.12·C`}</pre>
            <p className="fine">权重体现优先级：先能用，再快，最后才是稳。</p>
          </div>
        </div>
      </section>

      <section className="panel pad">
        <h2>2. 从分组到站点</h2>
        <p className="sub">
          每个轴单独聚合，不把三个轴揉成一个不可解释的数：
          <strong>80% 覆盖加权均值 + 20% 最差分组</strong>。权重是该分组在售的模型数——
          一个 40 模型的分组比 2 模型的分组更能代表这家站；同时最差分组仍占 20%，
          好分组掩盖不了坏分组。
        </p>
        <pre className="formula-block mono">{`axis(station) = 0.8 × Σ(wᵢ·axisᵢ)/Σwᵢ + 0.2 × min(axisᵢ)
score(station) = 0.62·A + 0.26·S + 0.12·C`}</pre>
        <p className="sub">
          举例：两家站点的可用性分别是 100 与 40，按 1:1 覆盖加权后是 70，再与最差分组混合得
          <code> 0.8×70 + 0.2×40 = 64</code>，不会因为一个满分分组就接近 100。
        </p>
      </section>

      <section className="panel pad">
        <h2 id="admission">3. 准入、排名与并列</h2>
        <ul className="limits">
          <li>分组在 7 天窗口内至少 <b>24 条</b>探针才计分；不足的分组在详情页标注「样本不足」。</li>
          <li>站点至少 <b>一半</b>的分组计分才进入榜单，否则进入「未排名」区。</li>
          <li>没有任何分组达到门槛时，页面显示 <b>未评分 / —</b>，不把 0 当作最差；此时仍可参考可用率、价格与模型覆盖。</li>
          <li>按综合分降序；同分时先比可用性分，再比稳定性分。</li>
          <li>分数保留一位小数，但排序用未取整的原始值，避免大量并列。</li>
          <li>新站上架 60 天内会在列表标记「新站」，但这不影响分数本身。</li>
        </ul>
      </section>

      <section className="panel pad">
        <h2>4. 不参与排名的两件事</h2>
        <div className="method-grid">
          <div>
            <h3>纯度（真实性）</h3>
            <p>
              只对 Claude 分组做思考签名与行为指纹抽检，约 3 天一次，取 7 天中位数。
              它衡量「是不是真模型」，和「稳不稳、快不快」是不同问题，因此单列一栏。
            </p>
          </div>
          <div>
            <h3>价格</h3>
            <p>
              价格 = 官方 USD 单价 × 站点结算汇率 × 分组倍率。便宜可能意味着更差的上游，
              所以价格单独展示，由读者自己权衡，不混进分数里做加权。
            </p>
          </div>
        </div>
        <table className="tier-table" style={{ marginTop: 14 }}>
          <thead><tr><th>区间</th><th>标签</th><th>含义</th></tr></thead>
          <tbody>
            <tr><td className="mono">≥ 80</td><td><span className="tier tier-official"><i className="tier-dot" />官方直连</span></td><td>签名交叉验证通过，行为与知识指纹吻合</td></tr>
            <tr><td className="mono">30–79</td><td><span className="tier tier-mixed"><i className="tier-dot" />混合线路</span></td><td>行为接近官方，但签名缺失或被替换</td></tr>
            <tr><td className="mono">&lt; 30</td><td><span className="tier tier-counterfeit"><i className="tier-dot" />疑似掺假</span></td><td>无有效签名且指纹不匹配</td></tr>
            <tr><td className="mono">无数据</td><td><span className="tier tier-unknown"><i className="tier-dot" />未检测</span></td><td>尚未抽检，或该分组不支持签名核验</td></tr>
          </tbody>
        </table>
      </section>

      <section className="panel pad">
        <h2>5. 已知局限</h2>
        <ul className="limits">
          <li>分数只描述我们用自己的 Key 探测到的那些分组；站点的其他分组、或针对特定客户的行为可能不同。</li>
          <li>探针短而便宜，能反映可达性与延迟，<strong>不能</strong>反映长输出吞吐与并发能力。</li>
          <li>Wilson 下界假设探针相互独立；真实探针存在时间相关性，因此它只是保守近似，不是严格统计保证。</li>
          <li>价格来自站点公开倍率，可能滞后；最终以站点当前页面的实际扣费为准。</li>
          <li>本站不做交易、不收款、不担保售后，不构成任何购买建议。</li>
        </ul>
        <p className="fine">
          对分数有异议？通过 <Link to="/submit">站点提交 / 认领</Link> 提交证据，我们会在 7 天内复核；
          复核结果会写进该站的变更记录。
        </p>
      </section>

      <section className="panel pad">
        <h2>6. 模型页的供给评分</h2>
        <p className="sub">
          模型档案页上的「供给评分」衡量的是<strong>现在买这个模型划不划算、供给稳不稳</strong>，不评价模型能力——
          能力需要公开基准，本站没有自测，所以不编造编程分或推理分。
        </p>
        <div className="method-grid">
          <div>
            <h3>五个维度</h3>
            <p>
              价格 0.35 · 覆盖 0.20 · 可用性 0.20 · 纯度 0.15 · 上下文 0.10，按可用维度归一。
              价格分 = 100 ×（1 − 最低价 ÷ 中位价 ÷ 2）；覆盖分 = 100 × √(报价站点数 ÷ 50)；
              可用性与纯度取报价站点的实测均值；上下文分按 log10(窗口) 映射。
            </p>
          </div>
          <div>
            <h3>为什么不用官方价算折扣</h3>
            <p>
              中转站实价 = 官方美元价 × 站点汇率 × 分组倍率，这三个因子都由站点自己定，
              直接算「比官方便宜多少」会把汇率与倍率混在一起。用「最低价 ÷ 中位价」只描述市场离散度，可复算也可解释。
            </p>
          </div>
        </div>
        <p className="sub" style={{ marginTop: 12 }}>
          数据边界：评分基于演示数据集里的报价与探针，接入真实站点与真实探针后同一套公式自动生效。
        </p>
      </section>

      <section className="panel pad">
        <h2>7. 参考与致谢</h2>
        <p className="sub">评分口径参考了以下公开项目的方法与实测经验，实现与数据完全独立：</p>
        <ul className="refs">
          {REFS.map((r) => (
            <li key={r.name}>
              <a href={r.url} target="_blank" rel="noreferrer noopener">{r.name}</a>
              <span>{r.desc}</span>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
