import { Button } from '@cloudflare/kumo/components/button';
import { Input } from '@cloudflare/kumo/components/input';
import { useMutation } from '@tanstack/react-query';
import { useId, useRef } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { CopyButton } from '../../components/text/CopyButton';
import { useT } from '../../i18n/locale';
import { checkAcme } from '../../lib/api/acme';
import { ApiError } from '../../lib/api/client';
import { useAcmeErrorText } from '../../lib/errors/acme';
import { domainFormMessages } from './domain-messages';

/** One read-only value of the record in mono, with Copy. */
function RecordValue({ label, value }: { label: string; value: string }) {
  const id = useId();
  const input = useRef<HTMLInputElement>(null);
  return (
    <div className="flex flex-col gap-1.5">
      <label id={`${id}-label`} htmlFor={id} className="text-sm font-medium text-kumo-strong">
        {label}
      </label>
      <div className="flex gap-2">
        <Input
          ref={input}
          id={id}
          aria-labelledby={`${id}-label`}
          readOnly
          dir="ltr"
          value={value}
          className="h-10 min-w-0 flex-1 font-mono text-sm pointer-coarse:h-9"
        />
        <CopyButton text={value} select={() => input.current?.select()} />
      </div>
    </div>
  );
}

/** The values Karvan saw where the record should be, from a 409 acme_dns_not_found. */
function seenOf(err: unknown): string[] {
  if (!(err instanceof ApiError) || err.code !== 'acme_dns_not_found') return [];
  const seen = err.detail.seen;
  return Array.isArray(seen) ? seen.filter((s): s is string => typeof s === 'string') : [];
}

/** The DNS-01 step: the TXT record to add, Check, and Ask anyway for split DNS. */
export function DnsStep({
  domainId,
  dns,
  onChecked,
}: {
  domainId: number;
  dns: { name: string; value: string };
  onChecked: () => void;
}) {
  const t = useT(domainFormMessages);
  const errorText = useAcmeErrorText();
  const check = useMutation({
    mutationFn: (force: boolean) => checkAcme(domainId, force),
    onSuccess: onChecked,
  });
  const notFound = check.error instanceof ApiError && check.error.code === 'acme_dns_not_found';
  const seen = seenOf(check.error);
  return (
    <div className="flex flex-col gap-4">
      <p>{t('dnsStep')}</p>
      <RecordValue label={t('dnsName')} value={dns.name} />
      <RecordValue label={t('dnsValue')} value={dns.value} />
      {check.isError ? (
        <InlineAlert tone={notFound ? 'warning' : 'danger'}>
          <span className="flex flex-col gap-1">
            <span>{errorText(check.error)}</span>
            {notFound ? (
              <span className="text-kumo-subtle">
                {seen.length > 0 ? (
                  <>
                    {t('seen', { values: '' })}
                    <bdi dir="ltr" className="font-mono text-[0.9em] break-all">
                      {seen.join(', ')}
                    </bdi>
                  </>
                ) : (
                  t('seenNone')
                )}
              </span>
            ) : null}
          </span>
        </InlineAlert>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <Button
          variant="primary"
          loading={check.isPending && check.variables === false}
          className="pointer-coarse:h-9"
          onClick={() => check.mutate(false)}
        >
          {t('check')}
        </Button>
        {notFound ? (
          <Button
            loading={check.isPending && check.variables}
            className="pointer-coarse:h-9"
            onClick={() => check.mutate(true)}
          >
            {t('checkAnyway')}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
