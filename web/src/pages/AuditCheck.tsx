import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api';
import type { AuditCheckDoc } from '../types';
import { Spinner } from '../components/ui';

export function AuditCheckPage() {
  const { key = '' } = useParams();
  const [doc, setDoc] = useState<AuditCheckDoc | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDoc(null);
    setError(null);
    api.auditCheck(key)
      .then((d) => {
        setDoc(d);
        document.title = `${d.label}：接口检测项目 | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [key]);

  if (error) {
    return (
      <main className="wrap">
        <div className="error-box">
          {error} <Link className="link-btn" to="/audit">返回接口检测</Link>
        </div>
      </main>
    );
  }
  if (!doc) {
    return <main className="wrap"><div className="loading-page"><Spinner /> 读取检测说明…</div></main>;
  }

  return (
    <main className="wrap">
      <div className="crumbs mono">
        <Link to="/audit">接口检测</Link><span>/</span><span className="dim">{doc.key}</span>
      </div>

      <section className="panel audit-doc">
        <span className="eyebrow"><i />AUDIT CHECK</span>
        <h1>{doc.label}</h1>
        <p className="audit-lead">{doc.short}</p>

        <h2>为什么重要</h2>
        <p>{doc.why}</p>

        <h2>怎么测</h2>
        <p>{doc.how}</p>

        <h2>判定标准</h2>
        <table className="index-table">
          <thead><tr><th>结果</th><th>条件</th></tr></thead>
          <tbody>
            <tr><td><span className="verdict-mini ok">通过</span></td><td>{doc.pass}</td></tr>
            <tr><td><span className="verdict-mini warn">观察</span></td><td>{doc.warn}</td></tr>
            <tr><td><span className="verdict-mini bad">失败</span></td><td>{doc.fail}</td></tr>
          </tbody>
        </table>

        <h2>局限</h2>
        <p>{doc.limit}</p>

        <div className="audit-doc-actions">
          <Link className="btn btn-primary btn-sm" to="/audit">打开检测工具 →</Link>
          <Link className="btn btn-ghost btn-sm" to="/methodology">评分方法</Link>
        </div>
      </section>
    </main>
  );
}
