import { useT } from '../../i18n/locale';
import { ApiError } from '../../lib/api/client';
import type { CopyReason } from '../../lib/api/certificates';
import { useErrorText } from '../../lib/errors/errors';
import { copyMessages } from './copy-messages';

const reasons: readonly CopyReason[] = [
  'not_found',
  'link',
  'outside',
  'foreign_file',
  'taken',
  'no_key',
  'no_name',
  'write_failed',
];

const isReason = (r: unknown): r is CopyReason =>
  typeof r === 'string' && (reasons as readonly string[]).includes(r);

/** A refused or failed copy in words, with the path it is about. */
export function useCopyReason(): (e: { reason: CopyReason; path: string }) => string {
  const t = useT(copyMessages);
  return (e) => t(`reason_${e.reason}`, { path: e.path });
}

/** Why a copy was refused, else the shared words for the failure. */
export function useCopyErrorText(): (err: unknown) => string {
  const reason = useCopyReason();
  const shared = useErrorText();
  return (err) => {
    if (err instanceof ApiError && err.code === 'copy_refused' && isReason(err.detail.reason)) {
      return reason({ reason: err.detail.reason, path: String(err.detail.path ?? '') });
    }
    return shared(err);
  };
}
