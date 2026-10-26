import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CalendarBlankIcon, CaretDownIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { rangeKeys, type RangeKey } from '../../lib/api/analytics';
import { analyticsMessages } from './messages';
/** The time range, like the dashboard's analytics header. The page is always live, so there is no refresh. */
export function RangeControl({
  range,
  onRange,
}: {
  range: RangeKey;
  onRange: (r: RangeKey) => void;
}) {
  const t = useT(analyticsMessages);
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="secondary"
            className="pointer-coarse:h-11"
            aria-label={`${t('rangeLabel')}: ${t(`range_${range}`)}`}
            icon={<CalendarBlankIcon size={16} />}
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
  );
}
