import '../styles/app.css';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { Wordmark } from '../components/brand/Wordmark';
import { LocaleProvider } from '../i18n/locale';
import { ThemeProvider } from '../lib/preferences/theme';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(
    <StrictMode>
      <ThemeProvider>
        <LocaleProvider>
          <main className="grid min-h-dvh place-items-center">
            <Wordmark size={40} />
          </main>
        </LocaleProvider>
      </ThemeProvider>
    </StrictMode>,
  );
}
