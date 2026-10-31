import { Button } from '@cloudflare/kumo/components/button';
import { LinkIcon } from '@phosphor-icons/react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { useT } from '../../i18n/locale';
import { batchKeys, getBatchGroup } from '../../lib/api/batches';
import type { JobRow } from '../../lib/api/migrations';
import { newBatchRulePath, newSitePath } from '../../lib/api/rules';
import { liveMessages } from './messages';

const button = 'pointer-coarse:h-9';

/**
 * What a finished migration offers for the old links. When a rule already serves its old panel,
 * the run's users are served by it with no change, so the rule opens; otherwise a new rule is
 * filled in from the run's old panel, or from the job for a run from before batches.
 */
export function OldLinksAction({ row }: { row: JobRow }) {
  const t = useT(liveMessages);
  const navigate = useNavigate();
  const batch = row.batch?.id ?? 0;
  const group = useQuery({
    queryKey: batchKeys.group(batch),
    queryFn: () => getBatchGroup(batch),
    enabled: batch > 0,
  });
  if (batch > 0 && group.isPending) return null;
  const rule = group.data?.rules[0];
  if (rule) {
    return (
      <>
        <Button
          variant="primary"
          icon={<LinkIcon size={16} />}
          className={button}
          onClick={() => void navigate(`/rules/${String(rule.id)}`)}
        >
          {t('openRule')}
        </Button>
        <p className="basis-full text-sm text-kumo-subtle">
          {t('oldLinksServed')}
          <bdi>{rule.name}</bdi>
        </p>
      </>
    );
  }
  return (
    <Button
      variant="primary"
      icon={<LinkIcon size={16} />}
      className={button}
      title={t('oldLinksHint')}
      onClick={() =>
        void navigate(
          batch > 0
            ? newBatchRulePath(batch)
            : newSitePath({
                panelId: row.panel.id,
                sourceKind: row.source.kind,
                sourceUrl: row.source.origin,
              }),
        )
      }
    >
      {t('oldLinks')}
    </Button>
  );
}
