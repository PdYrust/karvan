import type { StatusTone } from '../../components/feedback/StatusBadge';
import type { JobState } from '../../lib/api/migrations';
import type { MigrationsKey } from './messages';

/** How each job state reads: its status tone and its word. Both draft states are drafts to the operator. */
export const jobStates: Readonly<Record<JobState, { tone: StatusTone; word: MigrationsKey }>> = {
  created: { tone: 'neutral', word: 'stateDraft' },
  checked: { tone: 'neutral', word: 'stateDraft' },
  running: { tone: 'active', word: 'stateRunning' },
  done: { tone: 'success', word: 'stateDone' },
  failed: { tone: 'danger', word: 'stateFailed' },
  rolled_back: { tone: 'neutral', word: 'stateRolledBack' },
  interrupted: { tone: 'warning', word: 'stateInterrupted' },
};
