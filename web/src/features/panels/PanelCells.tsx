import { HardDrivesIcon } from '@phosphor-icons/react';
import { Link } from 'react-router';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { PanelMark } from '../../components/panels/PanelLabel';
import { panelNames } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import {
  databaseOf,
  engineName,
  panelKind,
  panelRelease,
  type Panel,
  type PanelControl,
} from '../../lib/api/panels';
import { controlMessages, controlText } from './control';
import { panelsMessages } from './messages';

/** The small tag next to the panel the switcher points at. */
export function CurrentTag() {
  const t = useT(panelsMessages);
  return (
    <span className="shrink-0 rounded-md px-1.5 text-tiny font-normal text-kumo-subtle ring ring-kumo-line">
      {t('current')}
    </span>
  );
}

/** The panel's logo and folder name, linking to its page. */
export function PanelName({ panel, current }: { panel: Panel; current: boolean }) {
  const kind = panelKind(panel);
  return (
    <span className="flex min-w-0 items-center gap-2">
      {kind ? (
        <PanelMark kind={kind} size={18} />
      ) : (
        <HardDrivesIcon size={18} aria-hidden="true" className="shrink-0 text-kumo-subtle" />
      )}
      <Link
        to={`/panels/${String(panel.id)}`}
        className="truncate font-medium text-kumo-strong underline decoration-kumo-line underline-offset-4 hover:decoration-current"
      >
        <bdi>{panel.name}</bdi>
      </Link>
      {current ? <CurrentTag /> : null}
    </span>
  );
}

/** Findings that mean the schema check refused the panel's database. */
const refusedCodes = new Set(['unknown_revision', 'schema_mismatch']);
/**
 * "PasarGuard 5.4.1". A revision the schema check accepted but that has no
 * name says so; "unsupported" is only for a database the check refused.
 */
export function PanelVersion({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const kind = panelKind(panel);
  if (!kind) return <span className="text-kumo-subtle">{t('unknownKind')}</span>;
  const release = panelRelease(panel.revision);
  if (!release && panel.revision && !panel.findings.some((f) => refusedCodes.has(f.code))) {
    return <span>{t(`unnamedVersion_${kind}`)}</span>;
  }
  // The name and its version are one Latin unit, so Persian keeps "PasarGuard 5.4.1" in that order.
  return (
    <span>
      <bdi dir="ltr" className="tabular">
        {release ? `${panelNames[kind]} ${release}` : panelNames[kind]}
      </bdi>
      {!release && panel.revision ? (
        <span className="ms-1.5 text-sm text-kumo-subtle">{t('unsupportedVersion')}</span>
      ) : null}
    </span>
  );
}
/** The engine, with the file, socket or host it lives at. Never a password: the address has none. */
export function PanelDatabase({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const { engine, where } = databaseOf(panel);
  return (
    <span className="flex min-w-0 flex-col">
      <span className={engine ? undefined : 'text-kumo-subtle'}>
        {engine ? engineName(engine) : t('notRead')}
      </span>
      {where ? (
        // self-start keeps a short host under the engine name in Persian too.
        <bdi
          dir="ltr"
          title={where}
          className="max-w-full self-start truncate font-mono text-tiny text-kumo-subtle"
        >
          {where}
        </bdi>
      ) : null}
    </span>
  );
}

/** Whether Karvan can stop and start the panel, or in a few words why not. */
export function ControlState({ control }: { control: PanelControl | undefined }) {
  const tc = useT(controlMessages);
  if (!control) return null;
  return (
    <StatusLabel tone={control.available ? 'success' : 'neutral'}>
      {tc(controlText(control).label)}
    </StatusLabel>
  );
}
