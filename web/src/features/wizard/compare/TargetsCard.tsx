import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { Checkbox } from '@cloudflare/kumo/components/checkbox';
import { InlineAlert } from '../../../components/feedback/InlineAlert';
import { Section } from '../../../components/layout/Section';
import { useT } from '../../../i18n/locale';
import type { Conflict, JobSummary } from '../../../lib/api/migrations';
import { setTargets } from '../../../lib/plan/plan';
import { protocolName } from '../protocols';
import { useWizard } from '../wizard-context';
import { conflictMessages } from './conflict-messages';
import { compareMessages } from './messages';

/** The groups or inbounds moved users get; the plan's own list, or the destination's defaults. */
export function TargetsCard({
  summary,
  conflicts,
}: {
  summary: JobSummary;
  conflicts: readonly Conflict[];
}) {
  const t = useT(compareMessages);
  const tc = useT(conflictMessages);
  const { plan } = useWizard();
  const own = plan.plan?.targets ?? null;
  const chosen = own ?? summary.userTargets ?? [];
  const pg = summary.destKind === 'pasarguard';
  const toggle = (id: string, on: boolean) => {
    const next = on ? [...chosen, id] : chosen.filter((x) => x !== id);
    plan.edit(setTargets(next));
  };
  return (
    <Section
      title={t('targetsTitle')}
      actions={
        own ? (
          <Button
            variant="ghost"
            size="sm"
            className="pointer-coarse:h-11"
            onClick={() => plan.edit(setTargets(null))}
          >
            {t('targetsDefault')}
          </Button>
        ) : null
      }
    >
      <div className="flex flex-col gap-3 px-4 pb-4">
        <p className="text-kumo-subtle">{t(pg ? 'targetsIntroGroups' : 'targetsIntroInbounds')}</p>
        {conflicts.some((c) => c.type === 'C09') ? (
          <InlineAlert>{tc('job_C09')}</InlineAlert>
        ) : null}
        {(summary.targets ?? []).length === 0 ? (
          <p className="text-kumo-subtle">{t(pg ? 'targetsNoGroups' : 'targetsNoInbounds')}</p>
        ) : (
          <ul className="flex flex-col rounded-lg ring ring-kumo-hairline">
            {(summary.targets ?? []).map((g) => (
              <li
                key={g.id}
                className="flex min-h-12 items-center gap-3 border-t border-kumo-hairline px-3 py-2 first:border-0"
              >
                <Checkbox
                  checked={chosen.includes(g.id)}
                  onCheckedChange={(on) => toggle(g.id, on)}
                  label={
                    <span className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                      <bdi className="text-kumo-strong">{g.name}</bdi>
                      <span className="text-sm text-kumo-subtle">
                        <bdi dir="ltr">{(g.protocols ?? []).map(protocolName).join(', ')}</bdi>
                      </span>
                      {g.disabled ? <Badge variant="outline">{t('targetDisabled')}</Badge> : null}
                    </span>
                  }
                />
                <span className="ms-auto shrink-0 text-sm text-kumo-subtle">
                  {g.hosts === 0 ? t('targetNoHosts') : t('targetHosts', { count: g.hosts })}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Section>
  );
}
