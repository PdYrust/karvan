import { Radio } from '@cloudflare/kumo/components/radio';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { CodeText } from '../../../components/text/CodeText';
import { useT } from '../../../i18n/locale';
import type { ReloadMethod, ReloadOption } from '../../../lib/api/preflight';
import { startMessages, type StartKey } from './messages';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';
const reasons: readonly string[] = ['not_built_in', 'no_full_admin', 'panel_unreadable'];
function reasonKey(r: string | undefined): StartKey {
  return r && reasons.includes(r) ? (`reason_${r}` as StartKey) : 'reason_unknown';
}
function effectKey(e: string): StartKey | null {
  const k = `effect_${e}`;
  return Object.hasOwn(startMessages.en, k) ? (k as StartKey) : null;
}
/** What one method does on this panel, as a short list; why it cannot run, when it cannot. */
function Effects({ o }: { o: ReloadOption }) {
  const t = useT(startMessages);
  const keys = o.effects.map(effectKey).filter((k): k is StartKey => k !== null);
  return (
    <span className="flex flex-col gap-1.5 text-sm text-kumo-subtle">
      {o.available ? null : <span className="text-kumo-default">{t(reasonKey(o.reason))}</span>}
      <span className="flex list-disc flex-col gap-0.5 ps-4">
        {keys.map((k) => (
          <span key={k} className="list-item">
            <CodeText text={t(k)} />
          </span>
        ))}
      </span>
      {o.command ? (
        <span className="flex flex-wrap items-center gap-1.5">
          {t('command')}
          <CodeText text={`\`${o.command}\``} />
        </span>
      ) : null}
    </span>
  );
}
/** The reload methods as radio cards, every one shown; one Karvan cannot run is disabled. */
export function ReloadChoice({
  options,
  value,
  onChange,
}: {
  options: { data?: ReloadOption[]; error: unknown; refetch: () => unknown };
  value: ReloadMethod | undefined;
  onChange: (m: ReloadMethod) => void;
}) {
  const t = useT(startMessages);
  const load = useFirstLoad(options.data);
  if (options.error) {
    return (
      <ErrorState
        message={t('reloadFailed')}
        error={options.error}
        onRetry={() => void options.refetch()}
      />
    );
  }
  if (!load.data) return <Bone className="h-48 w-full" />;
  return (
    <Reveal on={load.reveal}>
      <Radio.Group
        legend={t('reloadTitle')}
        description={t('reloadBody')}
        appearance="card"
        value={value ?? ''}
        onValueChange={(v: string) => onChange(v as ReloadMethod)}
      >
        {load.data.map((o) => (
          <Radio.Item
            key={o.method}
            value={o.method}
            disabled={!o.available}
            label={
              <span className="flex flex-col gap-1">
                <span className="font-medium text-kumo-strong">{t(`method_${o.method}`)}</span>
                <Effects o={o} />
              </span>
            }
          />
        ))}
      </Radio.Group>
    </Reveal>
  );
}
