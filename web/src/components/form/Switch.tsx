import { Switch as KumoSwitch, type SwitchProps } from '@cloudflare/kumo/components/switch';
import { cn } from '@cloudflare/kumo/utils';
/**
 * The one switch of the panel. Kumo slides the thumb with physical left
 * offsets, so in Persian the whole switch is mirrored: on puts the thumb at
 * the inline end, which is the left in right to left text.
 */
export function Switch({ className, ...props }: SwitchProps) {
  return (
    <KumoSwitch
      {...props}
      // The switch is 18 px high; the hit area reaches 44 px on touch screens.
      className={cn(
        "relative shrink-0 rtl:-scale-x-100 pointer-coarse:before:absolute pointer-coarse:before:-inset-x-[9px] pointer-coarse:before:-inset-y-[13px] pointer-coarse:before:content-['']",
        className,
      )}
    />
  );
}
