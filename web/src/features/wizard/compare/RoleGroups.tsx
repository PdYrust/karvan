import { WarningIcon } from '@phosphor-icons/react';
import { useT } from '../../../i18n/locale';
import type { Role } from '../../../lib/api/migrations';
import { protocolName } from '../protocols';
import { compareMessages } from './messages';

/** The enabled groups a role gives users, with their protocols and hosts; read only. */
export function RoleGroups({ role }: { role: Role }) {
  const t = useT(compareMessages);
  const groups = role.groups.filter((g) => !g.disabled);
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium text-kumo-strong">{t('roleGroups')}</h3>
      {role.allGroups ? <p className="text-sm text-kumo-subtle">{t('roleAllGroupsNote')}</p> : null}
      {groups.length === 0 ? (
        <p className="text-kumo-subtle">{t('roleNoGroups')}</p>
      ) : (
        <ul className="flex flex-col rounded-lg ring ring-kumo-hairline">
          {groups.map((g) => (
            <li
              key={g.id}
              className="flex min-h-11 flex-wrap items-center gap-x-3 gap-y-0.5 border-t border-kumo-hairline px-3 py-2 first:border-0"
            >
              <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
                <bdi className="text-kumo-strong">{g.name}</bdi>
                <span className="text-sm text-kumo-subtle">
                  <bdi dir="ltr">{(g.protocols ?? []).map(protocolName).join(', ')}</bdi>
                </span>
              </span>
              <span className="ms-auto flex shrink-0 items-center gap-1.5 text-sm text-kumo-subtle">
                {g.hosts === 0 ? (
                  <>
                    <WarningIcon size={16} className="text-kumo-warning" aria-hidden="true" />
                    {t('targetNoHosts')}
                  </>
                ) : (
                  t('targetHosts', { count: g.hosts })
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
