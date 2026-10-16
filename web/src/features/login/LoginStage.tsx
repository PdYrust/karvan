import { GirihPattern } from '../../components/brand/GirihPattern';
import { Logo3D } from '../../components/brand/logo3d/Logo3D';
import { useT } from '../../i18n/locale';
import { loginMessages } from './messages';

/** The one bold area of the sign-in page: tile lattice, the 3D star, one line of promise. */
export function LoginStage() {
  const t = useT(loginMessages);
  return (
    <section className="relative flex h-full flex-col items-center justify-center overflow-hidden bg-kumo-recessed px-6 py-8 lg:py-12">
      <GirihPattern className="pointer-events-none absolute inset-0 text-kumo-hairline [mask-image:radial-gradient(closest-side,currentColor,transparent)]" />
      <Logo3D className="size-44 sm:size-56 lg:size-[min(400px,52vh)]" />
      <div className="relative mt-2 hidden max-w-sm text-center text-balance lg:block">
        <p className="text-heading font-semibold text-kumo-strong">{t('tagline')}</p>
        <p className="mt-2 text-sm text-kumo-subtle">{t('taglineDetail')}</p>
      </div>
    </section>
  );
}
