import { useOverview } from '../../features/overview/use-overview';
import { useT } from '../../i18n/locale';
import { sharedMessages } from '../../i18n/shared';
import { shellMessages } from './messages';

const repo = 'https://github.com/PdYrust/karvan';
const link = 'hover:text-kumo-default hover:underline';

/** One quiet row under every page, like the dashboard's footer. */
export function Footer() {
  const t = useT(shellMessages);
  const ts = useT(sharedMessages);
  const version = useOverview().data?.version;
  return (
    <footer className="flex flex-wrap items-center gap-x-6 gap-y-2 border-t border-kumo-hairline px-4 py-4 text-sm text-kumo-subtle sm:px-8">
      {version ? (
        <span className="flex items-center gap-1.5">
          {ts('brand')}
          <bdi dir="ltr" className="tabular">
            {version}
          </bdi>
        </span>
      ) : null}
      <a className={link} href={repo} target="_blank" rel="noreferrer">
        {t('footerSource')}
      </a>
      <a className={link} href={`${repo}/security`} target="_blank" rel="noreferrer">
        {t('footerSecurity')}
      </a>
    </footer>
  );
}
