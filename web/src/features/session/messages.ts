import type { Messages } from '../../i18n/locale';

const en = {
  signOutFailed: 'Sign-out didn’t go through. You are still signed in; try again.',
};

type Key = keyof typeof en;

const fa: Record<Key, string> = {
  signOutFailed: 'خروج انجام نشد و هنوز وارد هستید. دوباره امتحان کنید.',
};

export const sessionMessages: Messages<Key> = { en, fa };
