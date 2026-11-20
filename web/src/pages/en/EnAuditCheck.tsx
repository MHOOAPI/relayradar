import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../../api';
import type { AuditCheckDoc } from '../../types';
import { Spinner } from '../../components/ui';

export function EnAuditCheckPage() {
  const { key = '' } = useParams();
  const [doc, setDoc] = useState<AuditCheckDoc | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDoc(null);
    setError(null);
    api.auditCheckEn(key)
      .then((d) => {
        setDoc(d);
        document.title = `${d.label} — audit check | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [key]);

  if (error) {
    return <main className="wrap"><div className="error-box">{error} <Link className="link-btn" to="/en/audit">Back to audit</Link></div></main>;
  }
  if (!doc) {
    return <main className="wrap"><div className="loading-page"><Spinner /> Loading check…</div></main>;
  }

  return (
    <main className="wrap">
      <div className="crumbs mono">
        <Link to="/en/audit">Audit</Link><span>/</span><span className="dim">{doc.key}</span>
      </div>
      <section className="panel audit-doc">
        <span className="eyebrow"><i />AUDIT CHECK</span>
        <h1>{doc.label}</h1>
        <p className="audit-lead">{doc.short}</p>

        <h2>Why it matters</h2>
        <p>{doc.why}</p>
        <h2>How we measure</h2>
        <p>{doc.how}</p>
        <h2>Verdicts</h2>
        <table className="index-table">
          <thead><tr><th>Result</th><th>Condition</th></tr></thead>
          <tbody>
            <tr><td><span className="verdict-mini ok">pass</span></td><td>{doc.pass}</td></tr>
            <tr><td><span className="verdict-mini warn">watch</span></td><td>{doc.warn}</td></tr>
            <tr><td><span className="verdict-mini bad">fail</span></td><td>{doc.fail}</td></tr>
          </tbody>
        </table>
        <h2>Limitations</h2>
        <p>{doc.limit}</p>

        <div className="audit-doc-actions">
          <Link className="btn btn-primary btn-sm" to="/en/audit">Open the audit tool →</Link>
          <Link className="btn btn-ghost btn-sm" to="/en/methodology">Methodology</Link>
        </div>
      </section>
    </main>
  );
}
