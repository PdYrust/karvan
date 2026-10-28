import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { compareStatus, useJobEvents } from '../../../lib/api/events';
import { sitePrefill } from '../../../lib/api/rules';
import {
  cancelJob,
  compareJob,
  createJob,
  migrationKeys,
  type NewJob,
} from '../../../lib/api/migrations';
import { errorOf } from '../../../lib/errors/errors';
import { useCurrentPanel } from '../../panels/current-panel';
import { usePanels } from '../../panels/use-panels';
import { stepPath } from '../steps';
import {
  certOf,
  defaultDestination,
  emptyValues,
  missingField,
  toNewJob,
  type CertFacts,
  type ConnectValues,
  type MissingField,
} from './form';

const jobIdPattern = /^[0-9a-f]{24}$/;

function prefilled(params: URLSearchParams): ConnectValues {
  const p = sitePrefill(params);
  return {
    ...emptyValues,
    panelId: p.panelId ?? 0,
    url: p.sourceUrl ?? '',
    kind: p.sourceKind ?? 'auto',
  };
}

/** The field each missing value focuses. */
const fieldIds: Partial<Record<MissingField, string>> = {
  url: 'source-url',
  username: 'source-user',
  password: 'source-pass',
  apiKey: 'source-key',
};

/**
 * Connect's state: the form, the job it made (kept in `?job=` so a reload
 * follows the same read), the read's events, and the one certificate the
 * operator chose to trust. The sign-in stays in component state (D10).
 */
export function useConnect() {
  const [params, setParams] = useSearchParams();
  const raw = params.get('job') ?? '';
  const jobId = jobIdPattern.test(raw) ? raw : undefined;
  const navigate = useNavigate();
  const qc = useQueryClient();
  const panels = usePanels().data ?? [];
  const { panel: current } = useCurrentPanel();
  const { stream, lost } = useJobEvents(jobId);
  const status = compareStatus(stream);

  // "Start again with the same address" fills in the destination, the address and the kind.
  const [typed, setTyped] = useState<ConnectValues>(() => prefilled(params));
  const values: ConnectValues = {
    ...typed,
    panelId: typed.panelId || defaultDestination(panels, current),
  };
  const [missing, setMissing] = useState<MissingField | null>(null);
  const [pin, setPin] = useState<string>();

  const compare = useMutation({ mutationFn: compareJob });
  const cancel = useMutation({ mutationFn: cancelJob });
  const create = useMutation({
    // The request carries the old panel's password: never keep it in the cache (D10).
    gcTime: 0,
    mutationFn: (job: NewJob) => createJob(job),
    onSuccess: ({ id }) => {
      void qc.invalidateQueries({ queryKey: migrationKeys.all });
      setParams({ job: id });
      compare.mutate(id);
    },
  });

  // Once the read and the build are done, Summary takes over; Back then returns to an empty form.
  useEffect(() => {
    if (jobId && status === 'built') void navigate(stepPath('summary', jobId), { replace: true });
  }, [jobId, status, navigate]);

  const createError = create.error ? errorOf(create.error) : null;
  const streamCert = stream.failure ? certOf(stream.failure.code, stream.failure.detail) : null;
  const createCert = createError ? certOf(createError.code, createError.detail) : null;
  const cert: CertFacts | null = (jobId ? streamCert : createCert) ?? null;

  const submit = (trusted = pin) => {
    const m = missingField(values);
    setMissing(m);
    if (m) {
      const id = fieldIds[m];
      if (id) document.getElementById(id)?.focus();
      return;
    }
    create.mutate(toNewJob(values, trusted));
  };

  return {
    jobId,
    stream,
    lost,
    status,
    values,
    set: (patch: Partial<ConnectValues>) => {
      // A trusted certificate belongs to the address it came from.
      if (patch.url !== undefined && patch.url !== typed.url) setPin(undefined);
      setMissing(null);
      setTyped((v) => ({ ...v, ...patch }));
    },
    missing,
    pin,
    cert,
    create,
    compare,
    cancel,
    submit: () => submit(),
    /** Trusts exactly this certificate and signs in again with it pinned. */
    trust: (c: CertFacts) => {
      setPin(c.sha256);
      create.reset();
      if (missingField(values)) setParams({});
      else submit(c.sha256);
    },
    readAgain: () => {
      if (jobId) compare.mutate(jobId);
    },
    stop: () => {
      if (jobId) cancel.mutate(jobId);
    },
    changeDetails: () => {
      create.reset();
      compare.reset();
      setParams({});
    },
  };
}
