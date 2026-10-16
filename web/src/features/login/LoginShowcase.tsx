import { LanguageSwitch } from '../../components/preferences/LanguageSwitch';
import { ThemeToggle } from '../../components/preferences/ThemeToggle';
import { PanelLogo, panelNames, type PanelKind } from '../../components/panels/PanelLogo';
import { useT } from '../../i18n/locale';
import globeUrl from './globe.svg';
import { routeFrom, routeTo } from './globe-points';
import { loginMessages } from './messages';

const onBrandButton = 'text-on-brand hover:bg-on-brand/15';

function Stop({ kind, at }: { kind: PanelKind; at: readonly [number, number] }) {
  return (
    <div
      className="absolute flex -translate-x-1/2 -translate-y-1/2 flex-col items-center gap-2"
      // The globe image is not mirrored in RTL, so its points are physical.
      style={{ left: `${at[0]}%`, top: `${at[1]}%` }}
    >
      <span className="grid size-14 place-items-center rounded-xl bg-on-brand text-(--color-kumo-brand) ring-4 ring-on-brand/25">
        <PanelLogo kind={kind} size={36} />
      </span>
      <span className="rounded-md bg-kumo-brand px-2 text-sm font-medium text-on-brand">
        {panelNames[kind]}
      </span>
    </div>
  );
}

/** The brand side of the sign-in page: the two panels Karvan moves users between. */
export function LoginShowcase() {
  const t = useT(loginMessages);
  return (
    <section className="relative hidden min-h-dvh flex-col overflow-hidden bg-kumo-brand text-on-brand lg:flex">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute start-[-5%] top-[40%] aspect-square w-[110%]"
      >
        <img src={globeUrl} alt="" loading="lazy" decoding="async" className="size-full" />
        <Stop kind="marzban" at={routeFrom} />
        <Stop kind="pasarguard" at={routeTo} />
      </div>

      <div className="relative flex justify-end gap-1 p-6">
        <LanguageSwitch className={onBrandButton} />
        <ThemeToggle className={onBrandButton} />
      </div>

      <div className="relative flex max-w-lg flex-col px-14 pt-4">
        <p dir="ltr" className="self-start font-mono text-sm text-on-brand/80">
          Marzban ⇄ PasarGuard
        </p>
        <h2 className="mt-3 text-[28px] leading-tight font-bold text-balance">{t('tagline')}</h2>
        <p className="mt-3 text-on-brand/85">{t('taglineDetail')}</p>
      </div>
    </section>
  );
}
