import { Button, RefreshButton } from '@cloudflare/kumo/components/button';
import { ButtonGroup } from '@cloudflare/kumo/components/button-group';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CaretDownIcon, ClockIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { rangeKeys, type RangeKey } from '../../lib/api/analytics';
import { analyticsMessages } from './messages';

/** Refresh and the time range, joined like the dashboard's analytics header. */
export function RangeControl({
  range,
  onRange,
  onRefresh,
  refreshing,
}: {
  range: RangeKey;
  onRange: (r: RangeKey) => void;
  onRefresh: () => void;
  refreshing: boolean;
}) {
  const t = useT(analyticsMessages);
  return (
    <ButtonGroup aria-label={t('rangeLabel')}>
      <RefreshButton
        variant="secondary"
        aria-label={t('refresh')}
        title={t('refresh')}
        loading={refreshing}
        onClick={onRefresh}
        className="pointer-coarse:size-11"
      />
      <DropdownMenu>
        <DropdownMenu.Trigger
          render={
            <Button
              variant="secondary"
              className="pointer-coarse:h-11"
              aria-label={`${t('rangeLabel')}: ${t(`range_${range}`)}`}
              icon={<ClockIcon size={16} />}
            >
              {t(`range_${range}`)}
              <CaretDownIcon size={12} aria-hidden="true" className="text-kumo-subtle" />
            </Button>
          }
        />
        <DropdownMenu.Content align="end" className="min-w-48">
          <DropdownMenu.RadioGroup value={range} onValueChange={(v: RangeKey) => onRange(v)}>
            {rangeKeys.map((k) => (
              <DropdownMenu.RadioItem key={k} value={k} closeOnClick>
                <span className="flex w-full items-center gap-2">
                  {t(`range_${k}`)}
                  <span className="ms-auto">
                    <DropdownMenu.RadioItemIndicator />
                  </span>
                </span>
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Content>
      </DropdownMenu>
    </ButtonGroup>
  );
}
