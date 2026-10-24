import type { Condition } from './plan';

/**
 * Ready-made categories: one click adds a category with these conditions
 * (api-migration.md 6.3). The key becomes the new category's id and names
 * its words in the page's messages.
 */
export const presetKeys = [
  'expired',
  'expiring',
  'no-expiry',
  'unlimited',
  'almost-used',
  'inactive',
  'never-connected',
  'disabled',
  'on-hold',
  'has-note',
] as const;

export type PresetKey = (typeof presetKeys)[number];

export const presetConditions: Readonly<Record<PresetKey, Condition>> = {
  expired: { field: 'expired', op: 'is', value: 'true' },
  expiring: { field: 'expires_within_days', op: 'lte', value: '7' },
  'no-expiry': { field: 'unlimited_time', op: 'is', value: 'true' },
  unlimited: { field: 'unlimited_traffic', op: 'is', value: 'true' },
  'almost-used': { field: 'used_percent', op: 'gte', value: '90' },
  inactive: { field: 'not_online_days', op: 'gte', value: '30' },
  'never-connected': { field: 'never_connected', op: 'is', value: 'true' },
  disabled: { field: 'status', op: 'in', list: ['disabled'] },
  'on-hold': { field: 'status', op: 'in', list: ['on_hold'] },
  'has-note': { field: 'has_note', op: 'is', value: 'true' },
};
