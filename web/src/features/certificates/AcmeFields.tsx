import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { useT } from '../../i18n/locale';
import { domainFormMessages } from './domain-messages';
import { linkClass } from './parts';

/** The subscriber agreement the first order of a directory accepts. */
const agreementUrl = 'https://letsencrypt.org/repository/';

/** The agreement Let's Encrypt asks for before the first order. */
export function AcmeFields({
  agreeTos,
  set,
}: {
  agreeTos: boolean;
  set: (patch: { agreeTos: boolean }) => void;
}) {
  const t = useT(domainFormMessages);
  return (
    <div className="flex flex-col items-start gap-1.5">
      <Checkbox
        label={t('agree')}
        checked={agreeTos}
        onCheckedChange={(on) => set({ agreeTos: on })}
      />
      <a
        href={agreementUrl}
        target="_blank"
        rel="noreferrer"
        className={`ms-6 text-sm ${linkClass}`}
      >
        {t('agreeLink')}
      </a>
    </div>
  );
}
