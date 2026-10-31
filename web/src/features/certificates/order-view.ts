import { isRunning, type AcmeError, type AcmeOrder, type AcmeState } from '../../lib/api/acme';

export type OrderStepId = 'start' | 'check' | 'issue' | 'done';
export interface OrderStep {
  id: OrderStepId;
  state: 'done' | 'current' | 'todo';
}

const stepIds: readonly OrderStepId[] = ['start', 'check', 'issue', 'done'];
const stepOf: Partial<Record<AcmeState, number>> = {
  starting: 0,
  waiting_dns: 1,
  validating: 1,
  issuing: 2,
  done: 4,
};

/** Where the order stands; an order that failed or was cancelled shows its words instead. */
export function orderSteps(o: AcmeOrder): OrderStep[] {
  const at = stepOf[o.state];
  if (at === undefined) return [];
  return stepIds.map((id, i) => {
    let state: OrderStep['state'] = 'todo';
    if (i < at) state = 'done';
    else if (i === at) state = 'current';
    return { id, state };
  });
}

/** The page reads a running order every 2 seconds. */
export function orderPoll(o: AcmeOrder | null | undefined): number | false {
  return isRunning(o) ? 2000 : false;
}

/** When Let's Encrypt takes requests again after a rate limit. */
export function retryTime(e: AcmeError | null | undefined): Date | undefined {
  return e?.retryAt ? new Date(e.retryAt) : undefined;
}
