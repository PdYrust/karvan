import { WarningCircleIcon } from '@phosphor-icons/react';
import { CodeText } from '../../components/text/CodeText';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import type { Finding } from '../../lib/api/panels';
import { findingDetail, findingMessages, findingText } from './findings';
import { panelsMessages } from './messages';

/** Each problem with a panel: what is wrong, how to fix it, and the value it is about. */
export function FindingList({ findings }: { findings: readonly Finding[] }) {
  const t = useT(panelsMessages);
  const tf = useT(findingMessages);
  return (
    <ul className="flex flex-col gap-4">
      {findings.map((f, i) => {
        const text = findingText(f.code);
        // A code without words still shows itself, so it can be looked up.
        const detail = text ? findingDetail(f) : f.code;
        return (
          <li key={`${f.code}:${String(i)}`} className="flex gap-3">
            <span className="flex h-lh items-center">
              <WarningCircleIcon
                size={18}
                weight="bold"
                aria-hidden="true"
                className="text-kumo-warning"
              />
            </span>
            <div className="flex min-w-0 flex-col gap-1">
              <p className="font-medium text-kumo-strong">
                {text ? <CodeText text={tf(text.what)} /> : t('problemOther')}
              </p>
              {text ? (
                <p className="text-sm text-kumo-subtle">
                  <CodeText text={tf(text.fix)} />
                </p>
              ) : null}
              {detail ? (
                <p className="text-kumo-default">
                  <Machine>{detail}</Machine>
                </p>
              ) : null}
            </div>
          </li>
        );
      })}
    </ul>
  );
}
