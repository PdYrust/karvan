import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import {
  ArrowSquareOutIcon,
  ChartLineIcon,
  CrosshairIcon,
  DotsThreeIcon,
} from '@phosphor-icons/react';
import { useNavigate } from 'react-router';
import { useT } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { useCurrentPanel } from './current-panel';
import { panelsMessages } from './messages';

/** The "..." menu of one panel row. */
export function PanelMenu({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const navigate = useNavigate();
  const { panel: current, select } = useCurrentPanel();
  const id = String(panel.id);
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={t('moreFor', { name: panel.name })}
            title={t('moreFor', { name: panel.name })}
            icon={<DotsThreeIcon size={20} weight="bold" />}
            className="pointer-coarse:size-9"
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-52">
        <DropdownMenu.Item className="gap-2" onClick={() => void navigate(`/panels/${id}`)}>
          <ArrowSquareOutIcon size={16} aria-hidden="true" className="rtl:-scale-x-100" />
          {t('open')}
        </DropdownMenu.Item>
        {current?.id === panel.id ? null : (
          <DropdownMenu.Item className="gap-2" onClick={() => select(panel.id)}>
            <CrosshairIcon size={16} aria-hidden="true" />
            {t('workOn')}
          </DropdownMenu.Item>
        )}
        <DropdownMenu.Item
          className="gap-2"
          onClick={() => void navigate(`/analytics/panels/${id}`)}
        >
          <ChartLineIcon size={16} aria-hidden="true" />
          {t('analytics')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}
