import { Toasty } from '@cloudflare/kumo/components/toast';
import { DirectionProvider } from '@cloudflare/kumo/primitives/direction-provider';
import { QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router';
import { OverviewPage } from '../features/overview/OverviewPage';
import { LocaleProvider, useLocale } from '../i18n/locale';
import { ApiError } from '../lib/api/client';
import { ThemeProvider } from '../lib/preferences/theme';
import { AppShell } from './shell/AppShell';
import { returnToSignIn } from '../features/session/use-session';

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    // A request that finds no session means it ended (timeout or sign-out
    // elsewhere); the server shows the sign-in page at the same address.
    onError: (err) => {
      if (err instanceof ApiError && err.status === 401) returnToSignIn();
    },
  }),
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
        <Directed>
          <QueryClientProvider client={queryClient}>
            <Toasty>
              <BrowserRouter basename={basename}>
                <Routes>
                  <Route element={<AppShell />}>
                    <Route index element={<OverviewPage />} />
                    <Route path="*" element={<Navigate to="/" replace />} />
                  </Route>
                </Routes>
              </BrowserRouter>
            </Toasty>
          </QueryClientProvider>
        </Directed>
      </LocaleProvider>
    </ThemeProvider>
  );
}
