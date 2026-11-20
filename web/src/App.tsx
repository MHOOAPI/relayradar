import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
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
import { AccountPage } from './pages/Account';
import { GoPage } from './pages/Go';
import { LoginPage } from './pages/Login';
import { RegisterPage } from './pages/Register';
import { VerifyPage } from './pages/Verify';
import { ForgotPage } from './pages/Forgot';
import { ResetPage } from './pages/Reset';
import { AdminPage } from './pages/Admin';
import { ModelLicensePage } from './pages/ModelLicense';
import { ModelComparePage } from './pages/ModelCompare';
import { StationsPage } from './pages/Stations';
import { FreeKeysPage } from './pages/FreeKeys';
import { NotFoundPage } from './pages/NotFound';
import { StatusPage } from './pages/Status';
import { ModelStatusPage } from './pages/ModelStatus';
import { ModelStatusDetailPage } from './pages/ModelStatusDetail';
import { ClaimEditPage } from './pages/ClaimEdit';
import { GuidesPage } from './pages/Guides';
import { GuidePage } from './pages/Guide';
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
          <Route path="account" element={<AccountPage />} />
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
          <Route path="leaderboards/window" element={<Navigate to="/leaderboards" replace />} />
          <Route path="sponsor" element={<Navigate to="/faq" replace />} />
          <Route path="status" element={<StatusPage />} />
          <Route path="model-status" element={<ModelStatusPage />} />
          <Route path="model-status/:key" element={<ModelStatusDetailPage />} />
          <Route path="claim/:token" element={<ClaimEditPage />} />
          <Route path="en/*" element={<Navigate to="/" replace />} />
          <Route path="guides" element={<GuidesPage />} />
          <Route path="guides/:slug" element={<GuidePage />} />
          <Route path="methodology" element={<MethodologyPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
