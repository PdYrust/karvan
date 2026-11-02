import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';
import { repoUrl } from './repo';

const iconBtn =
  'grid size-8 place-items-center rounded-md text-kumo-default hover:bg-kumo-tint pointer-coarse:size-9';

const githubPath =
  'M208.31,75.68A59.78,59.78,0,0,0,202.93,28,8,8,0,0,0,196,24a59.75,59.75,0,0,0-48,24H124A59.75,59.75,0,0,0,76,24a8,8,0,0,0-6.93,4,59.78,59.78,0,0,0-5.38,47.68A58.14,58.14,0,0,0,56,104v8a56.06,56.06,0,0,0,48.44,55.47A39.8,39.8,0,0,0,96,192v8H72a24,24,0,0,1-24-24A40,40,0,0,0,8,136a8,8,0,0,0,0,16,24,24,0,0,1,24,24,40,40,0,0,0,40,40H96v16a8,8,0,0,0,16,0V192a24,24,0,0,1,48,0v40a8,8,0,0,0,16,0V192a39.8,39.8,0,0,0-8.44-24.53A56.06,56.06,0,0,0,216,112v-8A58.14,58.14,0,0,0,208.31,75.68ZM200,112a40,40,0,0,1-40,40H112a40,40,0,0,1-40-40v-8a41.74,41.74,0,0,1,6.9-22.48A8,8,0,0,0,80,73.83a43.81,43.81,0,0,1,.79-33.58,43.88,43.88,0,0,1,32.32,20.06A8,8,0,0,0,119.82,64h32.35a8,8,0,0,0,6.74-3.69,43.87,43.87,0,0,1,32.32-20.06A43.81,43.81,0,0,1,192,73.83a8.09,8.09,0,0,0,1,7.65A41.72,41.72,0,0,1,200,104Z';

const telegramPath =
  'M228.88,26.19a9,9,0,0,0-9.16-1.57L17.06,103.93a14.22,14.22,0,0,0,2.43,27.21L72,141.45V200a15.92,15.92,0,0,0,10,14.83,15.91,15.91,0,0,0,17.51-3.73l25.32-26.26L165,220a15.88,15.88,0,0,0,10.51,4,16.3,16.3,0,0,0,5-.79,15.85,15.85,0,0,0,10.67-11.63L231.77,35A9,9,0,0,0,228.88,26.19Zm-61.14,36L78.15,126.35l-49.6-9.73ZM88,200V152.52l24.79,21.74Zm87.53,8L92.85,135.5l119-85.29Z';

/** The Telegram channel and GitHub repository links, next to the account button. */
export function GitHubLink() {
  const t = useT(shellMessages);
  return (
    <>
      <a
        href="https://t.me/PdYrust"
        target="_blank"
        rel="noopener noreferrer"
        aria-label={t('telegram')}
        title={t('telegram')}
        className={iconBtn}
      >
        <svg width={18} height={18} viewBox="0 0 256 256" fill="currentColor" aria-hidden="true">
          <path d={telegramPath} />
        </svg>
      </a>
      <a
        href={repoUrl}
        target="_blank"
        rel="noreferrer"
        aria-label={t('github')}
        title={t('github')}
        className={iconBtn}
      >
        <svg width={18} height={18} viewBox="0 0 256 256" fill="currentColor" aria-hidden="true">
          <path d={githubPath} />
        </svg>
      </a>
    </>
  );
}
