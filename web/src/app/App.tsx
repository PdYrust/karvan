import { Toasty } from '@cloudflare/kumo/components/toast';
import { DirectionProvider } from '@cloudflare/kumo/primitives/direction-provider';
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { lazy } from 'react';
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
const NotFoundPage = lazy(() =>
  import('../features/not-found/NotFoundPage').then((m) => ({ default: m.NotFoundPage })),
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
  return <DirectionProvider direction={dir}>{children}</DirectionProvider>;
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
                    <Route element={<AppShell />}>
                      <Route index element={<OverviewPage />} />
                      <Route path="settings" element={<SettingsPage />}>
                        <Route index element={<GeneralTab />} />
                        <Route path="safe-mode" element={<SafeModeTab />} />
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
