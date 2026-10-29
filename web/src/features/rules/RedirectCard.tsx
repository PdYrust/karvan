import { FactList } from '../../components/layout/FactList';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import { rulePath, type Redirect } from '../../lib/api/rules';
import { CertFactsView } from '../certificates/CertFactsView';
import { mono, Part, RuleCardFrame } from './card-parts';
import { rulesMessages } from './messages';

/** The host a redirect sends old links to, for its title row. */
function targetHost(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}

/** One redirect rule: what it matches and where it sends the old links (section 11). */
export function RedirectCard({
  redirect: r,
  domains,
  onDelete,
}: {
  redirect: Redirect;
  domains: readonly Domain[];
  onDelete: () => void;
}) {
  const t = useT(rulesMessages);
  const names = r.domains ?? [];
  const tls = r.https ? t('https') : t('plainHttp');
  const facts = [
    { label: t('domains'), value: names.length ? mono(names.join(', ')) : t('everyDomain') },
    { label: t('port'), value: mono(String(r.port)) },
    {
      label: t('path'),
      value: (
        <span>
          {mono(r.fromPath)} {r.match === 'prefix' ? t('andBelow') : t('exact')}
        </span>
      ),
    },
    { label: t('goesTo'), value: mono(r.toUrl) },
    { label: t('answersOver'), value: tls },
  ];
  return (
    <RuleCardFrame
      name={r.name}
      badge={
        <span className="flex min-w-0 items-center gap-2 text-sm text-kumo-subtle">
          <span className="font-medium text-kumo-default">{t('redirect')}</span>
          {mono(targetHost(r.toUrl))}
        </span>
      }
      summary={
        <p className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-sm text-kumo-subtle">
          <span className="min-w-0 truncate">
            {names.length ? mono(names.join(', ')) : t('everyDomain')}
          </span>
          <span>{mono(`:${String(r.port)}${r.fromPath}`)}</span>
          <span>{tls}</span>
        </p>
      }
      rule={r}
      domains={domains}
      to={rulePath({ kind: 'redirect', id: r.id })}
      onDelete={onDelete}
    >
      <div className="grid divide-kumo-hairline border-t border-kumo-hairline lg:grid-cols-2 lg:divide-x rtl:lg:divide-x-reverse">
        <div className="px-4 py-1">
          <FactList facts={facts} />
        </div>
        {r.certificate ? (
          <div className="border-t border-kumo-hairline lg:border-t-0">
            <Part title={t('certTitle')}>
              <CertFactsView facts={r.certificate.facts} />
            </Part>
          </div>
        ) : null}
      </div>
    </RuleCardFrame>
  );
}
