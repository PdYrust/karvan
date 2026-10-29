import { Button } from '@cloudflare/kumo/components/button';
import { DropdownMenu } from '@cloudflare/kumo/components/dropdown';
import { CaretDownIcon, CaretRightIcon } from '@phosphor-icons/react';
import { useId, useState, type ReactNode } from 'react';
import { Link, useNavigate } from 'react-router';
import { StatusLabel } from '../../components/feedback/StatusBadge';
import { DotsThreeIcon, PencilSimpleIcon, TrashIcon } from '../../components/icons/regular';
import { Collapse } from '../../components/layout/Collapse';
import { useT } from '../../i18n/locale';
import type { Domain } from '../../lib/api/domains';
import { rulesMessages, type RulesKey } from './messages';
import { warningLines, warningTone, type WarningLine } from './rule-view';

export const mono = (text: string) => (
  <bdi dir="ltr" className="font-mono text-[0.9em] break-all">
    {text}
  </bdi>
);

const linkClass =
  'text-kumo-default underline decoration-kumo-line underline-offset-4 hover:decoration-current';

/** A titled part of the card, split from the next by a hairline. */
export function Part({ title, children }: { title: string; children: ReactNode }) {
  const id = useId();
  return (
    <section aria-labelledby={id} className="flex flex-col gap-2 px-4 py-3">
      <h3 id={id} className="text-sm font-medium text-kumo-strong">
        {title}
      </h3>
      {children}
    </section>
  );
}

function RuleMenu({ name, to, onDelete }: { name: string; to: string; onDelete: () => void }) {
  const t = useT(rulesMessages);
  const navigate = useNavigate();
  const label = t('moreFor', { name });
  return (
    <DropdownMenu>
      <DropdownMenu.Trigger
        render={
          <Button
            variant="ghost"
            shape="square"
            aria-label={label}
            title={label}
            icon={<DotsThreeIcon size={20} />}
            className="shrink-0 pointer-coarse:size-9"
          />
        }
      />
      <DropdownMenu.Content align="end" className="min-w-44">
        <DropdownMenu.Item className="gap-2" onClick={() => void navigate(to)}>
          <PencilSimpleIcon size={16} />
          {t('edit')}
        </DropdownMenu.Item>
        <DropdownMenu.Item className="gap-2" variant="danger" onClick={onDelete}>
          <TrashIcon size={16} />
          {t('delete')}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu>
  );
}

const words: Partial<Record<WarningLine['code'], RulesKey>> = {
  cert_conflict: 'w_cert_conflict',
  domain_mismatch: 'w_domain_mismatch',
  expired: 'w_expired',
  expires_soon: 'w_expires_soon',
  renewal_failed: 'w_renewal_failed',
  renewal_waiting_dns: 'w_renewal_waiting_dns',
};

/** Its domains' warnings, each naming its domain as a link to that domain's certificate. */
function Warnings({ lines }: { lines: WarningLine[] }) {
  const t = useT(rulesMessages);
  if (lines.length === 0) return null;
  return (
    <ul className="flex flex-col gap-1 text-sm">
      {lines.map(({ domain, code }) => (
        <li key={`${String(domain?.id)} ${code}`}>
          <StatusLabel tone={warningTone(code)}>
            {domain ? (
              <>
                <Link to={`/domains/${String(domain.id)}/certificate`} className={linkClass}>
                  {mono(domain.name)}
                </Link>{' '}
                {t(words[code] ?? 'w_broken')}
              </>
            ) : (
              <>
                {t('notAdopted')}{' '}
                <Link to="/domains" className={linkClass}>
                  {t('domainsPage')}
                </Link>
              </>
            )}
          </StatusLabel>
        </li>
      ))}
    </ul>
  );
}

/**
 * One rule, closed to its title row, its summary and its warnings; the title
 * row opens the details in place.
 */
export function RuleCardFrame({
  name,
  badge,
  summary,
  rule,
  domains,
  to,
  onDelete,
  children,
}: {
  name: string;
  badge: ReactNode;
  summary: ReactNode;
  rule: { domains: string[] | null; warnings: string[] };
  domains: readonly Domain[];
  to: string;
  onDelete: () => void;
  children: ReactNode;
}) {
  const id = useId();
  const [open, setOpen] = useState(false);
  const Caret = open ? CaretDownIcon : CaretRightIcon;
  return (
    <article aria-labelledby={id} className="rounded-lg bg-kumo-base ring ring-kumo-hairline">
      <header className="flex items-start gap-3 py-2 ps-4 pe-2">
        {/* The name's button covers the title row, so a tap anywhere on it opens or closes. */}
        <div className="relative flex min-w-0 flex-1 items-start gap-3">
          <div className="flex min-w-0 flex-1 flex-col gap-1 py-1.5">
            <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
              <h2 id={id} className="min-w-0 font-semibold text-kumo-strong">
                <button
                  type="button"
                  aria-expanded={open}
                  aria-controls={`${id}-details`}
                  onClick={() => setOpen((o) => !o)}
                  className="text-start outline-none after:absolute after:inset-0 after:rounded-lg after:content-[''] focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-kumo-focus"
                >
                  <bdi>{name}</bdi>
                </button>
              </h2>
              {badge}
            </div>
            {summary}
          </div>
          <Caret
            size={16}
            aria-hidden="true"
            className="mt-2.5 shrink-0 text-kumo-subtle rtl:-scale-x-100"
          />
        </div>
        <RuleMenu name={name} to={to} onDelete={onDelete} />
      </header>
      <div className="flex flex-col empty:hidden px-4 pb-3 -mt-1">
        <Warnings lines={warningLines(rule.domains, rule.warnings, domains)} />
      </div>
      <Collapse id={`${id}-details`} open={open}>
        {children}
      </Collapse>
    </article>
  );
}
