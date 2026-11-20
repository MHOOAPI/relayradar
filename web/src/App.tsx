import { BrowserRouter, Link, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Home } from './pages/Home';
import { StationPage } from './pages/Station';
import { ModelsPage } from './pages/Models';
import { ModelPage } from './pages/ModelPage';
import { AuditPage } from './pages/Audit';
import { AuditCheckPage } from './pages/AuditCheck';
import { MethodologyPage } from './pages/Methodology';
import { LeaderboardsPage } from './pages/Leaderboards';
import { ComparePage } from './pages/Compare';
import { FreePage } from './pages/Free';
import { SubmitPage } from './pages/Submit';
import { ApiDocsPage } from './pages/ApiDocs';
import { ChangesPage } from './pages/Changes';
import { FaqPage } from './pages/Faq';
import { AlertsPage } from './pages/Alerts';
import { GoPage } from './pages/Go';
import { LoginPage } from './pages/Login';
import { RegisterPage } from './pages/Register';
import { VerifyPage } from './pages/Verify';
import { ForgotPage } from './pages/Forgot';
import { ResetPage } from './pages/Reset';
import { AdminPage } from './pages/Admin';
import { ModelLicensePage } from './pages/ModelLicense';
import { ModelComparePage } from './pages/ModelCompare';
import { WindowLeaderboardPage } from './pages/WindowLeaderboard';
import { EnHomePage } from './pages/en/EnHome';
import { EnMethodologyPage } from './pages/en/EnMethodology';
import { EnFaqPage } from './pages/en/EnFaq';
import { EnModelsPage } from './pages/en/EnModels';
import { EnModelPage } from './pages/en/EnModel';
import { EnGuidesPage } from './pages/en/EnGuides';
import { EnGuidePage } from './pages/en/EnGuide';
import { EnLeaderboardPage } from './pages/en/EnLeaderboard';
import { EnStationPage } from './pages/en/EnStation';
import { EnAuditPage } from './pages/en/EnAudit';
import { EnAuditCheckPage } from './pages/en/EnAuditCheck';
import { EnChangesPage } from './pages/en/EnChanges';
import { EnFreePage } from './pages/en/EnFree';
import { StationsPage } from './pages/Stations';
import { FreeKeysPage } from './pages/FreeKeys';
import { NotFoundPage } from './pages/NotFound';
import { StatusPage } from './pages/Status';
import { EnStatusPage } from './pages/en/EnStatus';
import { ModelStatusPage } from './pages/ModelStatus';
import { EnModelStatusPage } from './pages/en/EnModelStatus';
import { ModelStatusDetailPage } from './pages/ModelStatusDetail';
import { ClaimEditPage } from './pages/ClaimEdit';
import { EnModelStatusDetailPage } from './pages/en/EnModelStatusDetail';
import { SponsorPage } from './pages/Sponsor';
import { GuidesPage } from './pages/Guides';
import { GuidePage } from './pages/Guide';

function NotFound() {
  return (
    <main className="wrap">
      <div className="notfound">
        <span className="mono dim">404</span>
        <h1>页面不存在</h1>
        <Link className="btn btn-primary" to="/">回到榜单</Link>
      </div>
    </main>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="stations" element={<StationsPage />} />
          <Route path="stations/:slug" element={<StationPage />} />
          <Route path="models" element={<ModelsPage />} />
          <Route path="models/:id" element={<ModelPage />} />
          <Route path="leaderboards" element={<LeaderboardsPage />} />
          <Route path="compare" element={<ComparePage />} />
          <Route path="changes" element={<ChangesPage />} />
          <Route path="free" element={<FreePage />} />
          <Route path="free-api-keys" element={<FreeKeysPage />} />
          <Route path="audit" element={<AuditPage />} />
          <Route path="audit/:key" element={<AuditCheckPage />} />
          <Route path="submit" element={<SubmitPage />} />
          <Route path="api-docs" element={<ApiDocsPage />} />
          <Route path="faq" element={<FaqPage />} />
          <Route path="alerts" element={<AlertsPage />} />
          <Route path="go/:slug" element={<GoPage />} />
          <Route path="login" element={<LoginPage />} />
          <Route path="register" element={<RegisterPage />} />
          <Route path="verify" element={<VerifyPage />} />
          <Route path="forgot" element={<ForgotPage />} />
          <Route path="reset" element={<ResetPage />} />
          <Route path="admin" element={<AdminPage />} />
          <Route path="models/license" element={<ModelLicensePage />} />
          <Route path="models/license/:license" element={<ModelLicensePage />} />
          <Route path="compare/models" element={<ModelComparePage />} />
          <Route path="leaderboards/window" element={<WindowLeaderboardPage />} />
          <Route path="sponsor" element={<SponsorPage />} />
          <Route path="en" element={<EnHomePage />} />
          <Route path="en/methodology" element={<EnMethodologyPage />} />
          <Route path="en/faq" element={<EnFaqPage />} />
          <Route path="en/models" element={<EnModelsPage />} />
          <Route path="en/models/:id" element={<EnModelPage />} />
          <Route path="en/guides" element={<EnGuidesPage />} />
          <Route path="en/guides/:slug" element={<EnGuidePage />} />
          <Route path="en/leaderboards" element={<EnLeaderboardPage />} />
          <Route path="en/stations/:slug" element={<EnStationPage />} />
          <Route path="en/audit" element={<EnAuditPage />} />
          <Route path="en/audit/:key" element={<EnAuditCheckPage />} />
          <Route path="en/changes" element={<EnChangesPage />} />
          <Route path="en/free" element={<EnFreePage />} />
          <Route path="en/status" element={<EnStatusPage />} />
          <Route path="en/model-status" element={<EnModelStatusPage />} />
          <Route path="en/model-status/:key" element={<EnModelStatusDetailPage />} />
          <Route path="status" element={<StatusPage />} />
          <Route path="model-status" element={<ModelStatusPage />} />
          <Route path="model-status/:key" element={<ModelStatusDetailPage />} />
          <Route path="claim/:token" element={<ClaimEditPage />} />
          <Route path="*" element={<NotFoundPage />} />
          <Route path="guides" element={<GuidesPage />} />
          <Route path="guides/:slug" element={<GuidePage />} />
          <Route path="methodology" element={<MethodologyPage />} />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
