import { lazy } from 'react';

// The Settings page and its tabs, each loaded when it opens.
export const SettingsPage = lazy(() =>
  import('../features/settings/SettingsPage').then((m) => ({ default: m.SettingsPage })),
);
export const GeneralTab = lazy(() =>
  import('../features/settings/GeneralTab').then((m) => ({ default: m.GeneralTab })),
);
export const SafeModeTab = lazy(() =>
  import('../features/settings/SafeModeTab').then((m) => ({ default: m.SafeModeTab })),
);
export const AccessTab = lazy(() =>
  import('../features/settings/access/AccessTab').then((m) => ({ default: m.AccessTab })),
);
export const HttpsTab = lazy(() =>
  import('../features/https/HttpsTab').then((m) => ({ default: m.HttpsTab })),
);
export const TwoStepTab = lazy(() =>
  import('../features/settings/two-step/TwoStepTab').then((m) => ({ default: m.TwoStepTab })),
);
export const SessionsTab = lazy(() =>
  import('../features/settings/sessions/SessionsTab').then((m) => ({ default: m.SessionsTab })),
);
export const AuditTab = lazy(() =>
  import('../features/settings/audit/AuditTab').then((m) => ({ default: m.AuditTab })),
);
