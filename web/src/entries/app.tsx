import '../styles/app.css';
import { StrictMode, startTransition } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from '../app/App';
const root = document.getElementById('root');
if (root) {
  // A transition lets React yield while it builds the first screen, so the
  // shell and the page do not hold the main thread in one long task.
  startTransition(() => {
    createRoot(root).render(
      <StrictMode>
        <App />
      </StrictMode>,
    );
  });
}
