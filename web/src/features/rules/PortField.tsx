import { useQuery } from '@tanstack/react-query';
import { CodeText } from '../../components/text/CodeText';
import { useT } from '../../i18n/locale';
import { latinDigits } from '../../lib/format/number';
import { getPorts } from '../../lib/api/ports';
import { siteFormMessages } from './form-messages';
import { oldPort, portHint } from './port-hint';
import type { FieldsProps } from './SiteFields';
import { TextField } from './TextField';

/** The port a rule answers on, with a free port proposed when it is empty or taken (D14). */
export function PortField({ id, values, set, errorOf }: FieldsProps & { id: string }) {
  const t = useT(siteFormMessages);
  const prefer = oldPort(values.sourceUrl, values.domains);
  const ports = useQuery({ queryKey: ['ports', prefer], queryFn: () => getPorts(prefer) });
  const hint = portHint(ports.data, values.port);
  return (
    <TextField
      id={id}
      label={t('portLabel')}
      hint={
        hint ? (
          <CodeText
            text={t(hint.kind === 'free' ? 'portFree' : 'portTaken', {
              port: `\`${String(hint.kind === 'taken' ? hint.port : '')}\``,
              free: `\`${String(hint.free)}\``,
            })}
          />
        ) : undefined
      }
      error={errorOf('port')}
      value={values.port}
      onChange={(port) => set({ port: latinDigits(port).replace(/\D/g, '') })}
      machine
      inputMode="numeric"
    />
  );
}
