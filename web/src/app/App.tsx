import { Toasty } from '@cloudflare/kumo/components/toast';
import { CSPProvider } from '@cloudflare/kumo/primitives/csp-provider';
import { DirectionProvider } from '@cloudflare/kumo/primitives/direction-provider';
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { lazy, Suspense } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router';
import { OverviewPage } from '../features/overview/OverviewPage';
import { returnToSignIn } from '../features/session/use-session';
import { LocaleProvider, useLocale } from '../i18n/locale';
import { ApiError } from '../lib/api/client';
import { getSession } from '../lib/api/session';
import { RegionProvider } from '../lib/preferences/region';
import { ThemeProvider } from '../lib/preferences/theme';
import { AppShell } from './shell/AppShell';
import {
  SettingsPage,
  GeneralTab,
  SafeModeTab,
  AccessTab,
  HttpsTab,
  TwoStepTab,
  SessionsTab,
  AuditTab,
} from './settings-pages';

const NotFoundPage = lazy(() =>
  import('../features/not-found/NotFoundPage').then((m) => ({ default: m.NotFoundPage })),
);
const AnalyticsPage = lazy(() =>
  import('../features/analytics/AnalyticsPage').then((m) => ({ default: m.AnalyticsPage })),
);
const AnalyticsIndex = lazy(() =>
  import('../features/analytics/AnalyticsPage').then((m) => ({ default: m.AnalyticsIndex })),
);
const ServerTab = lazy(() =>
  import('../features/analytics/server/ServerTab').then((m) => ({ default: m.ServerTab })),
);
const PanelTab = lazy(() =>
  import('../features/analytics/databases/PanelTab').then((m) => ({ default: m.PanelTab })),
);
const PanelsPage = lazy(() =>
  import('../features/panels/PanelsPage').then((m) => ({ default: m.PanelsPage })),
);
const PanelPage = lazy(() =>
  import('../features/panels/PanelPage').then((m) => ({ default: m.PanelPage })),
);

const ConnectStep = lazy(() =>
  import('../features/wizard/connect/ConnectStep').then((m) => ({ default: m.ConnectStep })),
);
const JobWizard = lazy(() =>
  import('../features/wizard/JobWizard').then((m) => ({ default: m.JobWizard })),
);
const SummaryStep = lazy(() =>
  import('../features/wizard/summary/SummaryStep').then((m) => ({ default: m.SummaryStep })),
);
const CompareStep = lazy(() =>
  import('../features/wizard/compare/CompareStep').then((m) => ({ default: m.CompareStep })),
);
const ChangesStep = lazy(() =>
  import('../features/wizard/changes/ChangesStep').then((m) => ({ default: m.ChangesStep })),
);
const PreviewStep = lazy(() =>
  import('../features/wizard/preview/PreviewStep').then((m) => ({ default: m.PreviewStep })),
);
const StartStep = lazy(() =>
  import('../features/wizard/start/StartStep').then((m) => ({ default: m.StartStep })),
);
const CheckReportPage = lazy(() =>
  import('../features/wizard/preview/CheckReportPage').then((m) => ({
    default: m.CheckReportPage,
  })),
);
const MigrationsPage = lazy(() =>
  import('../features/migrations/MigrationsPage').then((m) => ({ default: m.MigrationsPage })),
);
const JobRedirect = lazy(() =>
  import('../features/migrations/JobRedirect').then((m) => ({ default: m.JobRedirect })),
);
const ReportPage = lazy(() =>
  import('../features/migrations/report/ReportPage').then((m) => ({ default: m.ReportPage })),
);
const ReportPrint = lazy(() =>
  import('../features/migrations/report/ReportPrint').then((m) => ({ default: m.ReportPrint })),
);
const BatchesPage = lazy(() =>
  import('../features/batches/BatchesPage').then((m) => ({ default: m.BatchesPage })),
);
const BatchPage = lazy(() =>
  import('../features/batches/BatchPage').then((m) => ({ default: m.BatchPage })),
);
const BackupsPage = lazy(() =>
  import('../features/backups/BackupsPage').then((m) => ({ default: m.BackupsPage })),
);
const DomainsPage = lazy(() =>
  import('../features/certificates/DomainsPage').then((m) => ({ default: m.DomainsPage })),
);
const DomainFormPage = lazy(() =>
  import('../features/certificates/DomainFormPage').then((m) => ({ default: m.DomainFormPage })),
);
const CertificateFormPage = lazy(() =>
  import('../features/certificates/CertificateFormPage').then((m) => ({
    default: m.CertificateFormPage,
  })),
);
const RulesPage = lazy(() =>
  import('../features/rules/RulesPage').then((m) => ({ default: m.RulesPage })),
);
const RuleFormPage = lazy(() =>
  import('../features/rules/RuleFormPage').then((m) => ({ default: m.RuleFormPage })),
);
const LivePage = lazy(() =>
  import('../features/live/LivePage').then((m) => ({ default: m.LivePage })),
);
/**
 * A request that finds no session means it ended (timeout or sign-out
 * elsewhere); the server shows the sign-in page at the same address. A write
 * answered "csrf" may mean the same, since CSRF is checked first.
 */
function onApiError(err: unknown) {
  if (!(err instanceof ApiError)) return;
  if (err.status === 401) returnToSignIn();
  if (err.status === 403 && err.code === 'csrf') {
    void getSession().then((s) => {
      if (!s.authenticated) returnToSignIn();
    }, returnToSignIn);
  }
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: onApiError }),
  mutationCache: new MutationCache({ onError: onApiError }),
  defaultOptions: {
    queries: {
      // Client errors will not fix themselves; server and network errors might.
      retry: (count, err) =>
        !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 2,
      refetchOnWindowFocus: false,
    },
  },
});

// Routes are relative to the hidden path the server wrote into <base>.
const basename = new URL(document.baseURI).pathname.replace(/\/$/, '');

function Directed({ children }: { children: React.ReactNode }) {
  const { dir } = useLocale();
  // The content security policy allows no inline styles, so Base UI's own
  // <style> element (it hides a select's scrollbar) is off; app.css has the rule.
  return (
    <CSPProvider disableStyleElements>
      <DirectionProvider direction={dir}>{children}</DirectionProvider>
    </CSPProvider>
  );
}

export function App() {
  return (
    <ThemeProvider>
      <LocaleProvider>
        <RegionProvider>
          <Directed>
            <QueryClientProvider client={queryClient}>
              <Toasty>
                <BrowserRouter basename={basename}>
                  <Routes>
                    {/* A printable page has no shell around it (DESIGN.md). */}
                    <Route
                      path="migrations/:id/preview/print"
                      element={
                        <Suspense>
                          <CheckReportPage />
                        </Suspense>
                      }
                    />
                    <Route
                      path="migrations/:id/report/print"
                      element={
                        <Suspense>
                          <ReportPrint />
                        </Suspense>
                      }
                    />
                    <Route element={<AppShell />}>
                      <Route index element={<OverviewPage />} />
                      <Route path="analytics" element={<AnalyticsPage />}>
                        <Route index element={<AnalyticsIndex />} />
                        <Route path="server" element={<ServerTab />} />
                        <Route path="panels/:id" element={<PanelTab />} />
                      </Route>
                      <Route path="migrations" element={<MigrationsPage />} />
                      <Route path="migrations/new" element={<ConnectStep />} />
                      <Route path="migrations/:id">
                        <Route index element={<JobRedirect />} />
                        <Route element={<JobWizard />}>
                          <Route path="summary" element={<SummaryStep />} />
                          <Route path="compare" element={<CompareStep />} />
                          <Route path="changes" element={<ChangesStep />} />
                          <Route path="preview" element={<PreviewStep />} />
                          <Route path="start" element={<StartStep />} />
                        </Route>
                        <Route path="live" element={<LivePage />} />
                        <Route path="report" element={<ReportPage />} />
                      </Route>
                      <Route path="batches" element={<BatchesPage />} />
                      <Route path="batches/:id" element={<BatchPage />} />
                      <Route path="backups" element={<BackupsPage />} />
                      <Route path="domains" element={<DomainsPage />} />
                      <Route path="domains/new" element={<DomainFormPage />} />
                      <Route path="domains/:id/certificate" element={<DomainFormPage />} />
                      <Route path="domains/certificates/:id" element={<CertificateFormPage />} />
                      <Route path="rules" element={<RulesPage />} />
                      <Route path="rules/new" element={<RuleFormPage />} />
                      <Route path="rules/:id" element={<RuleFormPage />} />
                      <Route path="rules/redirects/:id" element={<RuleFormPage redirect />} />
                      <Route path="panels" element={<PanelsPage />} />
                      <Route path="panels/:id" element={<PanelPage />} />
                      <Route path="settings" element={<SettingsPage />}>
                        <Route index element={<GeneralTab />} />
                        <Route path="safe-mode" element={<SafeModeTab />} />
                        <Route path="access" element={<AccessTab />} />
                        <Route path="https" element={<HttpsTab />} />
                        <Route path="two-step" element={<TwoStepTab />} />
                        <Route path="sessions" element={<SessionsTab />} />
                        <Route path="audit-log" element={<AuditTab />} />
                      </Route>
                      <Route path="*" element={<NotFoundPage />} />
                    </Route>
                  </Routes>
                </BrowserRouter>
              </Toasty>
            </QueryClientProvider>
          </Directed>
        </RegionProvider>
      </LocaleProvider>
    </ThemeProvider>
  );
}
