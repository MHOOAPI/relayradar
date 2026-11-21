import { lazy } from 'react';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Home } from './pages/Home';

// Home stays in the entry chunk for the fastest first paint; everything else is
// split per route so a visitor on the ranking page never downloads the admin
// console or the audit form.
const StationsPage = lazy(() => import('./pages/Stations').then((m) => ({ default: m.StationsPage })));
const StationPage = lazy(() => import('./pages/Station').then((m) => ({ default: m.StationPage })));
const ModelsPage = lazy(() => import('./pages/Models').then((m) => ({ default: m.ModelsPage })));
const ModelPage = lazy(() => import('./pages/ModelPage').then((m) => ({ default: m.ModelPage })));
const LeaderboardsPage = lazy(() => import('./pages/Leaderboards').then((m) => ({ default: m.LeaderboardsPage })));
const ComparePage = lazy(() => import('./pages/Compare').then((m) => ({ default: m.ComparePage })));
const ChangesPage = lazy(() => import('./pages/Changes').then((m) => ({ default: m.ChangesPage })));
const FreePage = lazy(() => import('./pages/Free').then((m) => ({ default: m.FreePage })));
const FreeKeysPage = lazy(() => import('./pages/FreeKeys').then((m) => ({ default: m.FreeKeysPage })));
const AuditPage = lazy(() => import('./pages/Audit').then((m) => ({ default: m.AuditPage })));
const AuditCheckPage = lazy(() => import('./pages/AuditCheck').then((m) => ({ default: m.AuditCheckPage })));
const SubmitPage = lazy(() => import('./pages/Submit').then((m) => ({ default: m.SubmitPage })));
const ApiDocsPage = lazy(() => import('./pages/ApiDocs').then((m) => ({ default: m.ApiDocsPage })));
const FaqPage = lazy(() => import('./pages/Faq').then((m) => ({ default: m.FaqPage })));
const AlertsPage = lazy(() => import('./pages/Alerts').then((m) => ({ default: m.AlertsPage })));
const AccountPage = lazy(() => import('./pages/Account').then((m) => ({ default: m.AccountPage })));
const GoPage = lazy(() => import('./pages/Go').then((m) => ({ default: m.GoPage })));
const LoginPage = lazy(() => import('./pages/Login').then((m) => ({ default: m.LoginPage })));
const RegisterPage = lazy(() => import('./pages/Register').then((m) => ({ default: m.RegisterPage })));
const VerifyPage = lazy(() => import('./pages/Verify').then((m) => ({ default: m.VerifyPage })));
const ForgotPage = lazy(() => import('./pages/Forgot').then((m) => ({ default: m.ForgotPage })));
const ResetPage = lazy(() => import('./pages/Reset').then((m) => ({ default: m.ResetPage })));
const AdminPage = lazy(() => import('./pages/Admin').then((m) => ({ default: m.AdminPage })));
const ModelLicensePage = lazy(() => import('./pages/ModelLicense').then((m) => ({ default: m.ModelLicensePage })));
const ModelComparePage = lazy(() => import('./pages/ModelCompare').then((m) => ({ default: m.ModelComparePage })));
const StatusPage = lazy(() => import('./pages/Status').then((m) => ({ default: m.StatusPage })));
const ModelStatusPage = lazy(() => import('./pages/ModelStatus').then((m) => ({ default: m.ModelStatusPage })));
const ModelStatusDetailPage = lazy(() => import('./pages/ModelStatusDetail').then((m) => ({ default: m.ModelStatusDetailPage })));
const ClaimEditPage = lazy(() => import('./pages/ClaimEdit').then((m) => ({ default: m.ClaimEditPage })));
const GuidesPage = lazy(() => import('./pages/Guides').then((m) => ({ default: m.GuidesPage })));
const GuidePage = lazy(() => import('./pages/Guide').then((m) => ({ default: m.GuidePage })));
const MethodologyPage = lazy(() => import('./pages/Methodology').then((m) => ({ default: m.MethodologyPage })));
const NotFoundPage = lazy(() => import('./pages/NotFound').then((m) => ({ default: m.NotFoundPage })));

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
