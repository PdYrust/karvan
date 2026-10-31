import { useQuery } from '@tanstack/react-query';
import { batchKeys, listBatchGroups } from '../../../lib/api/batches';
import { useT } from '../../../i18n/locale';
import { usedOrigins, type ConnectValues } from './form';
import { connectMessages } from './messages';

const used = { runs: 0, limit: 20 } as const;

/**
 * The old panel addresses finished migrations used, one tap each, so the same old panel is not
 * typed by its IP once and by its domain the next time: every run of it then joins one old panel.
 */
export function UsedAddresses({
  url,
  set,
}: {
  url: string;
  set: (patch: Partial<ConnectValues>) => void;
}) {
  const t = useT(connectMessages);
  const groups = useQuery({
    queryKey: batchKeys.groups(used),
    queryFn: () => listBatchGroups(used),
  });
  const origins = usedOrigins(groups.data?.groups ?? []);
  if (origins.length === 0) return null;
  return (
    <div className="flex flex-col gap-1.5">
      <span id="used-addresses" className="text-sm text-kumo-subtle">
        {t('usedBefore')}
      </span>
      <ul aria-labelledby="used-addresses" className="flex flex-wrap gap-2">
        {origins.map((o) => (
          <li key={o.origin} className="min-w-0">
            <button
              type="button"
              aria-pressed={url.trim() === o.origin}
              onClick={() => set({ url: o.origin, kind: o.kind })}
              className="inline-flex max-w-full items-center rounded-md bg-kumo-tint px-2.5 py-1 font-mono text-[0.85em] break-all text-kumo-default ring ring-kumo-hairline hover:bg-kumo-recessed aria-pressed:ring-kumo-brand pointer-coarse:min-h-9"
            >
              <bdi dir="ltr">{o.origin}</bdi>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
