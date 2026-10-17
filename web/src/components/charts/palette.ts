/** One of Kumo's five chart colors in tokens.css; series take them in order. */
export type SeriesColor = 1 | 2 | 3 | 4 | 5;

/** The CSS value of a chart color, for SVG fills and strokes. */
export function chartColor(c: SeriesColor): string {
  return `var(--color-chart-${String(c)})`;
}

export const gridColor = 'var(--color-chart-grid)';
