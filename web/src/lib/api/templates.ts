import type { StoredTemplate } from '../plan/changes';
import { api } from './client';

export const templateKeys = {
  all: ['templates'] as const,
};

/** Saved templates by name (api-resources.md 3.2). */
export async function listTemplates(): Promise<StoredTemplate[]> {
  const res = await api<{ templates: StoredTemplate[] | null }>('GET', 'templates');
  return res.templates ?? [];
}

/** Saves the reusable part of a job's saved plan under name; 409 template_name when it is taken. */
export function saveTemplate(jobId: string, name: string): Promise<StoredTemplate> {
  return api('POST', 'templates', { jobId, name });
}

/** A name the server takes: 1 to 64 bytes once trimmed. */
export function templateNameFits(name: string): boolean {
  const bytes = new TextEncoder().encode(name.trim()).length;
  return bytes > 0 && bytes <= 64;
}
