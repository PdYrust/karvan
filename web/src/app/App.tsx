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

const SettingsPage = lazy(() =>
  import('../features/settings/SettingsPage').then((m) => ({ default: m.SettingsPage })),
);
const GeneralTab = lazy(() =>
  import('../features/settings/GeneralTab').then((m) => ({ default: m.GeneralTab })),
);
const SafeModeTab = lazy(() =>
  import('../features/settings/SafeModeTab').then((m) => ({ default: m.SafeModeTab })),
);
const AccessTab = lazy(() =>
  import('../features/settings/access/AccessTab').then((m) => ({ default: m.AccessTab })),
);
const TwoStepTab = lazy(() =>
  import('../features/settings/two-step/TwoStepTab').then((m) => ({ default: m.TwoStepTab })),
);
const SessionsTab = lazy(() =>
  import('../features/settings/sessions/SessionsTab').then((m) => ({ default: m.SessionsTab })),
);
const AuditTab = lazy(() =>
  import('../features/settings/audit/AuditTab').then((m) => ({ default: m.AuditTab })),
);
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
const LinksPage = lazy(() =>
  import('../features/links/LinksPage').then((m) => ({ default: m.LinksPage })),
);
const SiteFormPage = lazy(() =>
  import('../features/links/SiteFormPage').then((m) => ({ default: m.SiteFormPage })),
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
                      <Route path="links" element={<LinksPage />} />
                      <Route path="links/new" element={<SiteFormPage />} />
                      <Route path="links/:id" element={<SiteFormPage />} />
                      <Route path="panels" element={<PanelsPage />} />
                      <Route path="panels/:id" element={<PanelPage />} />
                      <Route path="settings" element={<SettingsPage />}>
                        <Route index element={<GeneralTab />} />
                        <Route path="safe-mode" element={<SafeModeTab />} />
                        <Route path="access" element={<AccessTab />} />
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
