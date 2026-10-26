/**
 * The phone menu button's icon, as in the dashboard: three lines, the bottom
 * one shorter. Phosphor has none like it. The short line keeps to the start
 * edge, so the icon mirrors in RTL.
 */
export function MenuIcon({ size = 20 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      aria-hidden="true"
      className="rtl:-scale-x-100"
    >
      <path d="M4 6h16M4 12h16M4 18h9" />
    </svg>
  );
}
