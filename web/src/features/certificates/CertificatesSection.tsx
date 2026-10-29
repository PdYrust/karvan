import { useKumoToastManager } from '@cloudflare/kumo/components/toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { DataTable } from '../../components/data/DataTable';
import { ListToolbar } from '../../components/data/ListToolbar';
import { dataMessages } from '../../components/data/messages';
import { sortChoice } from '../../components/data/sort-choice';
import { TypeToConfirm } from '../../components/dialogs/TypeToConfirm';
import { useLocale, useT } from '../../i18n/locale';
import { certificateKeys, deleteCertificate, type Certificate } from '../../lib/api/certificates';
import { domainKeys } from '../../lib/api/domains';
import { useSubscriptionErrorText } from '../../lib/errors/subscriptions';
import { matchesTerm, sortRows, type ListSort } from '../../lib/lists/choices';
import { useLingering } from '../../lib/motion/use-lingering';
import { useNow } from '../overview/use-now';
import { certStateOf } from './cert-view';
import { CertRowCard, useCertColumns } from './CertRows';
import { ListFrame } from './ListFrame';
import { certificatesMessages } from './messages';

const states = ['all', 'active', 'attention', 'broken'] as const;
type StateFilter = (typeof states)[number];

/** The stored certificates as a list like Migrations, and the typed delete of one nothing uses. */
export function CertificatesSection({ certificates }: { certificates: readonly Certificate[] }) {
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
  const [asking, setAsking] = useState<Certificate | null>(null);
  // The dialog keeps the name while it fades out.
  const named = useLingering(asking);
  const remove = useMutation({
    mutationFn: (c: Certificate) => deleteCertificate(c.id),
    onSuccess: () => {
      setAsking(null);
      toasts.add({ title: t('certDeleted') });
      void qc.invalidateQueries({ queryKey: certificateKeys.all });
      void qc.invalidateQueries({ queryKey: domainKeys.all });
    },
  });
  const ask = (c: Certificate) => {
    remove.reset();
    setAsking(c);
  };
  const columns = useCertColumns(now, ask);
  const is = (c: Certificate, f: StateFilter) =>
    f === 'all' || certStateOf(c.facts.warnings).state === f;
  const terms = (c: Certificate) => [c.name, c.certFile, c.keyFile, ...c.facts.domains];
  const shown = sortRows(
    certificates.filter((c) => is(c, filter) && matchesTerm(query, terms(c))),
    sort,
    { name: (c) => c.name, time: (c) => c.id },
    locale,
  );
  const word = (f: StateFilter) => (f === 'all' ? td('showAll') : t(`state_${f}`));
  if (certificates.length === 0) {
    return (
      <section className="flex flex-col gap-3">
        <h2 className="text-heading font-semibold text-kumo-strong">{t('certsTitle')}</h2>
        <p className="rounded-lg bg-kumo-tint px-4 py-3 text-kumo-subtle">{t('certsEmpty')}</p>
      </section>
    );
  }
  return (
    <>
      <ListFrame
        title={t('certsTitle')}
        total={certificates.length}
        shown={shown.length}
        noMatch={t('noCertMatch')}
        toolbar={
          <ListToolbar
            search={{ label: t('searchCerts'), value: query, onChange: setQuery }}
            choices={[
              {
                label: td('filter'),
                value: filter,
                all: 'all',
                options: states.map((f) => ({
                  value: f,
                  label: word(f),
                  text: word(f),
                  count: certificates.filter((c) => is(c, f)).length,
                })),
                onChange: (f) => setFilter(f as StateFilter),
              },
              sortChoice(td, sort, setSort, ['name', 'newest']),
            ]}
          />
        }
      >
        <DataTable
          caption={t('certsTitle')}
          rows={shown}
          rowKey={(c) => c.id}
          columns={columns}
          card={(c) => <CertRowCard cert={c} now={now} onDelete={() => ask(c)} />}
        />
      </ListFrame>
      <TypeToConfirm
        open={asking !== null}
        onOpenChange={(open) => {
          if (!open) setAsking(null);
        }}
        title={t('deleteCertTitle')}
        word={named?.name ?? ''}
        action={t('delete')}
        pending={remove.isPending}
        error={remove.isError ? `${t('deleteCertFailed')} ${errorText(remove.error)}` : undefined}
        onConfirm={() => asking && remove.mutate(asking)}
      >
        {t('deleteCertBody')}
      </TypeToConfirm>
    </>
  );
}
