import '../styles/app.css';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { LoginPage } from '../features/login/LoginPage';
import { LocaleProvider } from '../i18n/locale';
import { ThemeProvider } from '../lib/preferences/theme';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(
    <StrictMode>
      <ThemeProvider>
        <LocaleProvider>
          <LoginPage />
        </LocaleProvider>
      </ThemeProvider>
    </StrictMode>,
  );
}
