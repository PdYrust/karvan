import '../styles/app.css';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { firstCheck } from '../features/login/first-check';
import { LoginPage } from '../features/login/LoginPage';
import { enterApp } from '../features/login/use-sign-in';
import { LocaleProvider } from '../i18n/locale';
import { ThemeProvider } from '../lib/preferences/theme';

const root = document.getElementById('root');
if (root) {
  // Nothing is painted until the session answers: the page stays the
  // background colour, a live session goes straight back in, and only a
  // signed-out one shows the form.
  void firstCheck().then(({ signedIn, error }) => {
    if (signedIn) {
      enterApp();
      return;
    }
    createRoot(root).render(
      <StrictMode>
        <ThemeProvider>
          <LocaleProvider>
            <LoginPage error={error} />
          </LocaleProvider>
        </ThemeProvider>
      </StrictMode>,
    );
  });
}
