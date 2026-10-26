import { GithubLogoIcon } from '@phosphor-icons/react';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';
import { repoUrl } from './repo';
/** The project's repository, next to the account button. */
export function GitHubLink() {
  const t = useT(shellMessages);
  return (
    <a
      href={repoUrl}
      target="_blank"
      rel="noreferrer"
      aria-label={t('github')}
      title={t('github')}
      className="grid size-8 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint pointer-coarse:size-11"
    >
      <GithubLogoIcon size={18} aria-hidden="true" />
    </a>
  );
}
