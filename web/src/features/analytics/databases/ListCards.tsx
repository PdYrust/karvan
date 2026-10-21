import { cn } from '@cloudflare/kumo/utils';
import {
  ArrowsClockwiseIcon,
  CheckCircleIcon,
  MinusCircleIcon,
  QuestionIcon,
  WarningCircleIcon,
  WarningIcon,
  type Icon,
} from '@phosphor-icons/react';
import { BarList } from '../../../components/charts/BarList';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import type { PanelData, PanelDetail, RangeKey } from '../../../lib/api/analytics';
import type { PanelKind } from '../../../lib/api/panels';
import { useFormat } from '../../../lib/preferences/region';
import { ChartCard } from '../ChartCard';
import { analyticsMessages, type AnalyticsKey } from '../messages';
import { databaseRange } from '../range';

function Quiet({ children }: { children: string }) {
  return <p className="text-sm text-kumo-subtle">{children}</p>;
}

/** The panel's largest tables by size, with their rows. */
export function TablesCard({
  detail,
  missing,
}: {
  detail: PanelDetail | undefined;
  missing: string | undefined;
}) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  return (
    <ChartCard title={t('tablesTitle')}>
      {missing || !detail ? (
        <Quiet>{missing ?? ''}</Quiet>
      ) : (
        <BarList
          label={t('tablesTitle')}
          items={detail.tables.map((x) => ({
            key: x.name,
            label: <Machine>{x.name}</Machine>,
            hint: t('tableRows', { count: x.rows }),
            value: x.bytes,
            display: fmt.bytes(x.bytes),
          }))}
        />
      )}
    </ChartCard>
  );
}

/** Admins (resellers) by how many users they own. */
export function AdminsCard({ data }: { data: PanelData }) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  return (
    <ChartCard title={t('adminsTitle')} aside={fmt.number(data.admins)}>
      <BarList
        label={t('adminsTitle')}
        items={data.topAdmins.map((a) => ({
          key: a.username,
          label: <Machine>{a.username}</Machine>,
          hint: t('adminHint', { active: a.active, traffic: fmt.bytes(a.usedTraffic) }),
          value: a.users,
          display: t('adminUsers', { count: a.users }),
        }))}
      />
    </ChartCard>
  );
}

/** Node states both panels report; status is an icon and a word, never a color alone. */
const nodeStates: Record<string, { icon: Icon; tone: string }> = {
  connected: { icon: CheckCircleIcon, tone: 'text-kumo-success' },
  connecting: { icon: ArrowsClockwiseIcon, tone: 'text-kumo-subtle' },
  error: { icon: WarningCircleIcon, tone: 'text-kumo-danger' },
  disabled: { icon: MinusCircleIcon, tone: 'text-kumo-subtle' },
  limited: { icon: WarningIcon, tone: 'text-kumo-warning' },
};

function NodeState({ status }: { status: string }) {
  const t = useT(analyticsMessages);
  const look = nodeStates[status];
  const Glyph = look?.icon ?? QuestionIcon;
  return (
    <span className="flex items-center gap-1.5">
      <Glyph
        size={14}
        weight="bold"
        aria-hidden="true"
        className={cn('shrink-0', look?.tone ?? 'text-kumo-subtle')}
      />
      {look ? t(`node_${status}` as AnalyticsKey) : <Machine>{status}</Machine>}
    </span>
  );
}

/** Each node with its state and the traffic it carried. */
export function NodesCard({ data }: { data: PanelData }) {
  const t = useT(analyticsMessages);
  const fmt = useFormat();
  return (
    <ChartCard title={t('nodesTitle')} aside={fmt.number(data.nodes.length)}>
      {data.nodes.length === 0 ? (
        <Quiet>{t('noNodes')}</Quiet>
      ) : (
        <BarList
          label={t('nodesTitle')}
          items={data.nodes.map((n) => ({
            key: n.name,
            label: <bdi>{n.name}</bdi>,
            hint: <NodeState status={n.status} />,
            value: n.uplink + n.downlink,
            display: t('nodeTraffic', { traffic: fmt.bytes(n.uplink + n.downlink) }),
          }))}
        />
      )}
    </ChartCard>
  );
}

/** Apps that fetch subscriptions: by update for PasarGuard, by user for Marzban. */
export function ClientsCard({
  detail,
  missing,
  kind,
  range,
}: {
  detail: PanelDetail | undefined;
  missing: string | undefined;
  kind: PanelKind | undefined;
  range: RangeKey;
}) {
  const t = useT(analyticsMessages);
  const r = databaseRange(range);
  const within = t(r === '1h' ? 'in_24h' : `in_${r}`);
  const unit = kind === 'pasarguard' ? 'clientUpdates' : 'clientUsers';
  const name = (n: string) =>
    n === 'other' || n === 'Browser' ? t(`client_${n}`) : <bdi dir="ltr">{n}</bdi>;
  const clients = detail?.clients ?? [];
  return (
    <ChartCard title={t('clientsTitle')}>
      {missing ? (
        <Quiet>{missing}</Quiet>
      ) : clients.length === 0 ? (
        <Quiet>{t('noClients', { range: within })}</Quiet>
      ) : (
        <BarList
          label={t('clientsTitle')}
          items={clients.map((c) => ({
            key: c.name,
            label: name(c.name),
            value: c.count,
            display: t(unit, { count: c.count }),
          }))}
        />
      )}
    </ChartCard>
  );
}
