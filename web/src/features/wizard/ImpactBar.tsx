import { CheckCircleIcon, WarningCircleIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import type { Impact } from '../../lib/api/migrations';
import { useFormat } from '../../lib/preferences/region';
import { impactCounts } from './impact';
import { wizardMessages } from './messages';

/** What the plan does now (the last save's impact) and what still stands in the way. */
export function ImpactBar({
  impact,
  blockers,
  saving,
  view = 'plan',
}: {
  impact: Impact | undefined;
  blockers: number;
  saving: boolean;
  /** The preview splits new users from those moved before. */
  view?: 'plan' | 'preview';
}) {
  const t = useT(wizardMessages);
  const fmt = useFormat();
  return (
    <section
      aria-label={t('impactLabel')}
      className="flex flex-col gap-4 rounded-lg bg-kumo-base px-4 py-4 ring ring-kumo-hairline"
    >
      <dl
        className={
          view === 'plan'
            ? 'grid grid-cols-2 gap-4 sm:grid-cols-4'
            : 'grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6'
        }
      >
        {impactCounts(impact, view).map(([k, n]) => (
          <div key={k} className="flex min-w-0 flex-col">
            <dt className="text-sm text-kumo-subtle">{t(k)}</dt>
            <dd className="text-heading font-semibold text-kumo-strong tabular">{fmt.number(n)}</dd>
          </div>
        ))}
      </dl>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-kumo-hairline pt-3 text-sm">
        <p className="flex items-center gap-1.5" aria-live="polite">
          {blockers > 0 ? (
            <WarningCircleIcon
              size={16}
              weight="bold"
              aria-hidden="true"
              className="shrink-0 text-kumo-danger"
            />
          ) : (
            <CheckCircleIcon
              size={16}
              weight="bold"
              aria-hidden="true"
              className="shrink-0 text-kumo-success"
            />
          )}
          <span>{blockers > 0 ? t('blockers', { count: blockers }) : t('noBlockers')}</span>
        </p>
        {saving ? (
          <span role="status" className="text-kumo-subtle">
            {t('saving')}
          </span>
        ) : null}
      </div>
    </section>
  );
}
