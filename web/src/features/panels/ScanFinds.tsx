import { FolderSimpleIcon } from '@phosphor-icons/react';
import { Section } from '../../components/layout/Section';
import { PanelMark } from '../../components/panels/PanelLabel';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import { panelKind, type Panel } from '../../lib/api/panels';
import { FindingList } from './FindingList';
import { panelsMessages } from './messages';

/** Folders the last scan saw but could not register, each with what to fix. */
export function ScanFinds({ finds }: { finds: readonly Panel[] }) {
  const t = useT(panelsMessages);
  return (
    <Section title={t('findsTitle')}>
      <p className="px-4 pb-4 text-sm text-kumo-subtle">{t('findsBody')}</p>
      <ul className="flex flex-col border-t border-kumo-hairline">
        {finds.map((p) => {
          const kind = panelKind(p);
          return (
            <li
              key={p.envPath}
              className="flex flex-col gap-3 border-b border-kumo-hairline px-4 py-4 last:border-b-0"
            >
              <span className="flex min-w-0 items-center gap-2">
                {kind ? (
                  <PanelMark kind={kind} size={18} />
                ) : (
                  <FolderSimpleIcon
                    size={18}
                    aria-hidden="true"
                    className="shrink-0 text-kumo-subtle"
                  />
                )}
                <Machine>{p.dir}</Machine>
              </span>
              <FindingList findings={p.findings} />
            </li>
          );
        })}
      </ul>
    </Section>
  );
}
