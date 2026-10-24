import { Badge } from '@cloudflare/kumo/components/badge';
import { WarningIcon } from '@phosphor-icons/react';
import { Section } from '../../../components/layout/Section';
import { useT } from '../../../i18n/locale';
import type { JobSummary } from '../../../lib/api/migrations';
import { protocolName } from '../protocols';
import { summaryMessages } from './messages';

/** The destination's groups (PasarGuard) or inbounds (Marzban): protocols, hosts, and where users go. */
export function TargetList({ summary }: { summary: JobSummary }) {
  const t = useT(summaryMessages);
  const pg = summary.destKind === 'pasarguard';
  const targets = summary.targets ?? [];
  const chosen = new Set(summary.userTargets ?? []);
  return (
    <Section title={t(pg ? 'groups' : 'inbounds')}>
      {targets.length === 0 ? (
        <p className="px-4 pb-4 text-kumo-subtle">{t(pg ? 'noGroups' : 'noInbounds')}</p>
      ) : (
        <ul className="flex flex-col">
          {targets.map((g) => (
            <li
              key={g.id}
              className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-kumo-hairline px-4 py-3"
            >
              <span className="flex min-w-0 flex-col gap-0.5">
                <span className="flex flex-wrap items-center gap-2">
                  <bdi className="font-medium text-kumo-strong">{g.name}</bdi>
                  {chosen.has(g.id) ? <Badge variant="secondary">{t('byDefault')}</Badge> : null}
                  {g.disabled ? <Badge variant="outline">{t('disabled')}</Badge> : null}
                </span>
                <span className="text-sm text-kumo-subtle">
                  {(g.protocols ?? []).map((p, i) => (
                    <bdi key={p} dir="ltr">
                      {i > 0 ? ', ' : ''}
                      {protocolName(p)}
                    </bdi>
                  ))}
                </span>
              </span>
              <span className="ms-auto flex items-center gap-1.5 text-sm text-kumo-subtle">
                {g.hosts === 0 ? (
                  <>
                    <WarningIcon size={16} className="text-kumo-warning" aria-hidden="true" />
                    {t('noHosts')}
                  </>
                ) : (
                  t('hosts', { count: g.hosts })
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}
