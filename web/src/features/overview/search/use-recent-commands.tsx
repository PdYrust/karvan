import { GearSixIcon, type Icon } from '@phosphor-icons/react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import type { Command, CommandGroup } from '../../../app/shell/commands';
import { shellMessages } from '../../../app/shell/messages';
import { nav } from '../../../app/shell/nav';
import { recentPage, type RecentPage } from '../../../app/shell/recent-pages';
import { PanelMark } from '../../../components/panels/PanelLabel';
import { useT } from '../../../i18n/locale';
import { panelKind, type Panel } from '../../../lib/api/panels';
import { readRecents } from '../../../lib/preferences/recents';
import { overviewMessages } from '../messages';
import { recentPanelOf, recentSections, recentTab } from '../recent-words';
const icon = 'size-[18px]';
/** Each section's sidebar icon; settings live in the account menu, with the gear. */
function sectionIcon(page: RecentPage): Icon {
  return nav.flatMap((g) => g.items).find((i) => i.to === `/${page.section}`)?.icon ?? GearSixIcon;
}
/**
 * The pages this browser opened last, as search rows: the page's name, and
 * its section as the quiet hint. A section's own page is named by the section alone.
 */
export function useRecentCommands(panels: readonly Panel[]): CommandGroup {
  const t = useT(overviewMessages);
  const ts = useT(shellMessages);
  const navigate = useNavigate();
  // Read once: the list only changes when the operator leaves this page.
  const [paths] = useState(readRecents);
  const items = paths.flatMap((path): Command[] => {
    const page = recentPage(path);
    if (!page) return [];
    const id = recentPanelOf(page);
    const panel = panels.find((p) => p.id === id);
    if (id !== undefined && !panel) return [];
    const section = ts(recentSections[page.section]);
    let name: string | undefined;
    if (page.section === 'settings') name = t(recentTab[page.tab]);
    else if (page.section === 'migrations' && page.page) name = t('newMigration');
    else if (page.section === 'rules' && page.page) name = t('newRule');
    else if (page.section === 'domains' && page.page) name = t('newDomain');
    else if (panel) name = panel.name;
    else if (page.section === 'analytics') name = t('recentServer');
    const kind = panel ? panelKind(panel) : undefined;
    const Glyph = sectionIcon(page);
    return [
      {
        id: `recent:${path}`,
        title: name ?? section,
        hint: name ? section : undefined,
        icon: kind ? <PanelMark kind={kind} size={18} /> : <Glyph className={icon} />,
        run: () => void navigate(path),
      },
    ];
  });
  return { id: 'recents', label: t('recentsTitle'), items };
}
