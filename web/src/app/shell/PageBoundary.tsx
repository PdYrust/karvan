import { Button } from '@cloudflare/kumo/components/button';
import { Component, type ReactNode } from 'react';
import { InlineAlert } from '../../components/feedback/InlineAlert';
import { useT } from '../../i18n/locale';
import { shellMessages } from './messages';

function PageFailed() {
  const t = useT(shellMessages);
  return (
    <div className="flex flex-col items-start gap-3 p-4 sm:p-8">
      <InlineAlert>{t('pageFailed')}</InlineAlert>
      <Button className="h-9 pointer-coarse:h-11" onClick={() => window.location.reload()}>
        {t('reload')}
      </Button>
    </div>
  );
}

/**
 * Catches a page that fails to render, such as a chunk that a newer
 * Karvan build replaced, so the shell stays usable.
 */
export class PageBoundary extends Component<
  { children: ReactNode; page: string },
  { failed: boolean }
> {
  override state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  override componentDidUpdate(prev: { page: string }) {
    if (prev.page !== this.props.page && this.state.failed) this.setState({ failed: false });
  }

  override render() {
    return this.state.failed ? <PageFailed /> : this.props.children;
  }
}
