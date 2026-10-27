import { cn } from '@cloudflare/kumo/utils';
import { RefreshButton } from '../actions/RefreshButton';
import { segment } from './segment';
/** Refresh as the last segment of a list toolbar. */
export function ToolbarRefresh({
  label,
  loading,
  onClick,
}: {
  label: string;
  loading: boolean;
  onClick: () => void;
}) {
  return (
    <RefreshButton
      variant="ghost"
      aria-label={label}
      title={label}
      loading={loading}
      onClick={onClick}
      className={cn(segment, 'size-auto h-full w-9 justify-center rounded-none')}
    />
  );
}
