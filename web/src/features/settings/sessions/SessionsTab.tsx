import { Badge } from '@cloudflare/kumo/components/badge';
import { Button } from '@cloudflare/kumo/components/button';
import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { DataTable, type Column } from '../../../components/data/DataTable';
import { ErrorState } from '../../../components/feedback/ErrorState';
import { Bone } from '../../../components/feedback/PageSkeleton';
import { Machine } from '../../../components/text/Machine';
import { useT } from '../../../i18n/locale';
import {
  accountKeys,
  endOtherSessions,
  endSession,
  getSessions,
  type AccountSession,
} from '../../../lib/api/account';
import { useErrorText } from '../../../lib/errors/errors';
import { readAgent } from '../../../lib/format/user-agent';
import { useFormat } from '../../../lib/preferences/region';
import { useNow } from '../../overview/use-now';
import { returnToSignIn } from '../../session/use-session';
import { sessionsMessages } from './messages';
import { Reveal } from '../../../components/feedback/Reveal';
import { useFirstLoad } from '../../../lib/motion/use-first-load';

const ltr = (name: string) => (
  <bdi key={name} dir="ltr">
    {name}
  </bdi>
);

/** "Chrome on Windows" as text for labels, and with the product names kept left to right for the page. */
function useAgentName() {
  const t = useT(sessionsMessages);
  const text = (ua: string) => {
    const { browser, system } = readAgent(ua);
    if (browser && system) return t('on', { browser, system });
    return browser || system || t('unknownBrowser');
  };
  const node = (ua: string): ReactNode => {
    const { browser, system } = readAgent(ua);
    if (!browser && !system) return t('unknownBrowser');
    if (!browser || !system) return ltr(browser || system);
    // Marks stand in for the names, so the sentence keeps its own word order in each language.
    return t('on', { browser: '%b', system: '%s' })
      .split(/(%b|%s)/)
      .map((part) => (part === '%b' ? ltr(browser) : part === '%s' ? ltr(system) : part));
  };
  return { text, node };
}

function When({ at, now }: { at: string; now?: Date }) {
  const fmt = useFormat();
  const d = new Date(at);
  return (
    <time dateTime={d.toISOString()} title={fmt.dateTime(d)} className="whitespace-nowrap">
      {now ? fmt.relative(d, now) : fmt.dateTime(d)}
    </time>
  );
}

const sessionsEvery = 30_000;

/** Every browser signed in now, the current one first; sign out one or all the others. */
export function SessionsTab() {
  const t = useT(sessionsMessages);
  const errorText = useErrorText();
  const agent = useAgentName();
  const fmt = useFormat();
  const now = useNow();
  const toasts = useKumoToastManager();
  const queryClient = useQueryClient();
  // A session that ends elsewhere (signed out, timed out) leaves the list on its own.
  const sessions = useQuery({
    queryKey: accountKeys.sessions,
    queryFn: getSessions,
    refetchInterval: sessionsEvery,
    refetchOnWindowFocus: true,
  });
  const refresh = () => queryClient.invalidateQueries({ queryKey: accountKeys.sessions });
  const failed = (err: unknown) => toasts.add({ title: errorText(err), variant: 'error' });
  const endOne = useMutation({
    mutationFn: (s: AccountSession) => endSession(s.id),
    onSuccess: (_, s) => {
      if (s.current) return returnToSignIn();
      toasts.add({ title: t('signedOutOne') });
      void refresh();
    },
    onError: failed,
  });
  const endOthers = useMutation({
    mutationFn: endOtherSessions,
    onSuccess: (count) => {
      toasts.add({ title: t('signedOutOthers', { count }) });
      void refresh();
    },
    onError: failed,
  });

  const load = useFirstLoad(sessions.data);
  if (sessions.isPending || load.holding) {
    return (
      <div role="status" aria-label={t('caption')} className="flex flex-col gap-3">
        <Bone className="h-10 w-64" />
        <Bone className="h-48 w-full" />
      </div>
    );
  }
  if (sessions.isError) {
    return (
      <ErrorState
        message={t('loadFailed')}
        error={sessions.error}
        onRetry={() => void sessions.refetch()}
      />
    );
  }
  const rows = sessions.data.toSorted((a, b) => Number(b.current) - Number(a.current));
  const others = rows.some((s) => !s.current);
  const name = (s: AccountSession) => (
    <span className="flex flex-wrap items-center gap-2">
      <span className="font-medium text-kumo-strong">{agent.node(s.userAgent)}</span>
      {s.current ? <Badge variant="info">{t('thisBrowser')}</Badge> : null}
    </span>
  );
  const signOut = (s: AccountSession) => (
    <Button
      size="sm"
      className="pointer-coarse:h-11"
      aria-label={t('signOutNamed', {
        name: agent.text(s.userAgent),
        time: fmt.dateTime(new Date(s.createdAt)),
      })}
      loading={endOne.isPending && endOne.variables.id === s.id}
      onClick={() => endOne.mutate(s)}
    >
      {t('signOut')}
    </Button>
  );
  const columns: Column<AccountSession>[] = [
    { key: 'browser', header: t('browser'), cell: name },
    { key: 'ip', header: t('ip'), cell: (s) => <Machine>{s.ip}</Machine> },
    { key: 'started', header: t('started'), cell: (s) => <When at={s.createdAt} /> },
    { key: 'last', header: t('lastUsed'), cell: (s) => <When at={s.lastSeenAt} now={now} /> },
    {
      key: 'actions',
      header: t('actions'),
      hideHeader: true,
      cell: signOut,
      className: 'text-end',
    },
  ];

  return (
    <Reveal on={load.reveal}>
      <section aria-labelledby="sessions-title" className="flex flex-col gap-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex flex-col gap-1">
            <h2 id="sessions-title" className="text-heading font-semibold text-kumo-strong">
              {t('title')}
            </h2>
            <p className="max-w-prose text-sm text-kumo-subtle">{t('description')}</p>
          </div>
          <Button
            className="shrink-0 self-start pointer-coarse:h-11"
            disabled={!others}
            title={others ? undefined : t('noOthers')}
            loading={endOthers.isPending}
            onClick={() => endOthers.mutate()}
          >
            {t('signOutOthers')}
          </Button>
        </div>
        <DataTable
          caption={t('caption')}
          rows={rows}
          rowKey={(s) => s.id}
          columns={columns}
          card={(s) => (
            <div className="flex flex-col gap-3 rounded-lg bg-kumo-base p-4 ring ring-kumo-hairline">
              {name(s)}
              <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
                <dt className="text-kumo-subtle">{t('ip')}</dt>
                <dd className="text-end">
                  <Machine>{s.ip}</Machine>
                </dd>
                <dt className="text-kumo-subtle">{t('started')}</dt>
                <dd className="text-end">
                  <When at={s.createdAt} />
                </dd>
                <dt className="text-kumo-subtle">{t('lastUsed')}</dt>
                <dd className="text-end">
                  <When at={s.lastSeenAt} now={now} />
                </dd>
              </dl>
              <div className="flex justify-end">{signOut(s)}</div>
            </div>
          )}
        />
      </section>
    </Reveal>
  );
}
