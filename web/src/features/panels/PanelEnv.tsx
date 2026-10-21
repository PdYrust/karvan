import { FactList, type Fact } from '../../components/layout/FactList';
import { Section } from '../../components/layout/Section';
import { CodeText } from '../../components/text/CodeText';
import { Machine } from '../../components/text/Machine';
import { useT } from '../../i18n/locale';
import type { Panel } from '../../lib/api/panels';
import { panelsMessages } from './messages';

// An IPv6 host needs brackets before the port.
function address(host: string, port: number): string {
  return `${host.includes(':') ? `[${host}]` : host}:${String(port)}`;
}

/** The values Karvan read from the panel's .env, as the panel itself reads them. No secret is among them. */
export function PanelEnv({ panel }: { panel: Panel }) {
  const t = useT(panelsMessages);
  const s = panel.settings;
  const unread = panel.findings.some((f) => f.code === 'env_unreadable');
  const facts: (Fact | false | undefined)[] = [
    { label: t('factEnv'), value: <Machine>{panel.envPath}</Machine> },
    !unread && {
      label: t('envDb'),
      value: panel.db ? (
        <Machine>{panel.db}</Machine>
      ) : (
        <span className="text-kumo-subtle">{t('notRead')}</span>
      ),
    },
    s.xrayConfig !== undefined && { label: t('envXray'), value: <Machine>{s.xrayConfig}</Machine> },
    !!s.excludeInbounds?.length && {
      label: t('envExclude'),
      value: <Machine>{s.excludeInbounds.join(' ')}</Machine>,
    },
    !unread && { label: t('envSubPath'), value: <Machine>{`/${s.subPath}`}</Machine> },
    s.subUrlPrefix !== undefined && {
      label: t('envSubPrefix'),
      value: <Machine>{s.subUrlPrefix}</Machine>,
    },
    !unread &&
      (s.uds
        ? { label: t('envSocket'), value: <Machine>{s.uds}</Machine> }
        : { label: t('envAddress'), value: <Machine>{address(s.host, s.port)}</Machine> }),
    s.certFile !== undefined && { label: t('envCert'), value: <Machine>{s.certFile}</Machine> },
    s.keyFile !== undefined && { label: t('envKey'), value: <Machine>{s.keyFile}</Machine> },
  ];
  return (
    <Section title={<CodeText text={t('envTitle')} />}>
      <div className="px-4 pb-2">
        <FactList facts={facts.filter((f): f is Fact => Boolean(f))} />
      </div>
    </Section>
  );
}
