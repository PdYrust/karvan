import { useEffect } from 'react';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';

/** "Settings · Karvan" in the browser tab. */
export function usePageTitle(title: string) {
  const ts = useT(sharedMessages);
  const brand = ts('brand');
  useEffect(() => {
    document.title = title ? `${title} | ${brand}` : brand;
  }, [title, brand]);
}
