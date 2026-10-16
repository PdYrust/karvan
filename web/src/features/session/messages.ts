import type { Messages } from '../../i18n/locale';

const en = {
  signOutFailed: "Couldn't sign out. Try again.",
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  signOutFailed: 'خروج انجام نشد. دوباره امتحان کنید.',
};

export const sessionMessages: Messages<Key> = { en, fa };
