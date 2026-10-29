import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { DataTable } from '../../components/data/DataTable';
import { ListToolbar } from '../../components/data/ListToolbar';
import { dataMessages } from '../../components/data/messages';
import { sortChoice } from '../../components/data/sort-choice';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { useLocale, useT } from '../../i18n/locale';
import { certificateKeys } from '../../lib/api/certificates';
import { deleteDomain, domainKeys, type Domain } from '../../lib/api/domains';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { matchesTerm, sortRows, type ListSort } from '../../lib/lists/choices';
import { useLingering } from '../../lib/motion/use-lingering';
import { useNow } from '../overview/use-now';
import { domainStateOf, type DomainState } from './cert-view';
import { DomainRowCard, useDomainColumns } from './DomainRows';
import { ListFrame } from './ListFrame';
import { certificatesMessages } from './messages';

const states = ['all', 'active', 'plain', 'attention', 'broken'] as const;
type StateFilter = (typeof states)[number];

/** Every domain rules can answer, as a list like Migrations; problems open from each state (D50, D61). */
export function DomainsSection({ domains }: { domains: readonly Domain[] }) {
  const t = useT(certificatesMessages);
  const td = useT(dataMessages);
  const { locale } = useLocale();
  const now = useNow();
  const errorText = useSubscriptionErrorText();
  const toasts = useKumoToastManager();
  const qc = useQueryClient();
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<StateFilter>('all');
  const [sort, setSort] = useState<ListSort>('name');
  const [asking, setAsking] = useState<Domain | null>(null);
  // The dialog keeps the name while it fades out.
  const named = useLingering(asking);
  const remove = useMutation({
    mutationFn: (d: Domain) => deleteDomain(d.id),
    onSuccess: () => {
      setAsking(null);
      toasts.add({ title: t('domainDeleted') });
      void qc.invalidateQueries({ queryKey: domainKeys.all });
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
    },
  });
  const ask = (d: Domain) => {
    remove.reset();
    setAsking(d);
  };
  const columns = useDomainColumns(now, ask);
  const stateOf = (d: Domain): DomainState => domainStateOf(d, now).state;
  const is = (d: Domain, f: StateFilter) => f === 'all' || stateOf(d) === f;
  const terms = (d: Domain) => [d.name, d.certificate?.name, ...d.rules.map((r) => r.name)];
  const shown = sortRows(
    domains.filter((d) => is(d, filter) && matchesTerm(query, terms(d))),
    sort,
    { name: (d) => d.name, time: (d) => d.id },
    locale,
  );
  const word = (f: StateFilter) => (f === 'all' ? td('showAll') : t(`state_${f}`));
  return (
    <>
      <ListFrame
        title={t('domainsTitle')}
        total={domains.length}
        shown={shown.length}
        noMatch={t('noDomainMatch')}
        toolbar={
          <ListToolbar
            search={{ label: t('searchDomains'), value: query, onChange: setQuery }}
            choices={[
              {
                label: td('filter'),
                value: filter,
                all: 'all',
                options: states.map((f) => ({
                  value: f,
                  label: word(f),
                  text: word(f),
                  count: domains.filter((d) => is(d, f)).length,
                })),
                onChange: (f) => setFilter(f as StateFilter),
              },
              sortChoice(td, sort, setSort, ['name', 'newest']),
            ]}
          />
        }
      >
        <DataTable
          caption={t('domainsTitle')}
          rows={shown}
          rowKey={(d) => d.id}
          columns={columns}
          card={(d) => <DomainRowCard domain={d} now={now} onDelete={() => ask(d)} />}
        />
      </ListFrame>
      <TypeToConfirm
        open={asking !== null}
        onOpenChange={(open) => {
          if (!open) setAsking(null);
        }}
        title={t('deleteDomainTitle')}
        word={named?.name ?? ''}
        action={t('delete')}
        pending={remove.isPending}
        error={remove.isError ? `${t('deleteDomainFailed')} ${errorText(remove.error)}` : undefined}
        onConfirm={() => asking && remove.mutate(asking)}
      >
        {t('deleteDomainBody')}
      </TypeToConfirm>
    </>
  );
}
