/** A browser and its system read from a User-Agent header; either is empty when unknown. */
export interface Agent {
  browser: string;
  system: string;
}

// Order matters: Edge, Opera and Samsung Internet also say Chrome, and Chrome says Safari.
const browsers: [RegExp, string][] = [
  [/\bEdg(?:e|A|iOS)?\//, 'Edge'],
  [/\b(?:OPR|Opera)\//, 'Opera'],
  [/\bSamsungBrowser\//, 'Samsung Internet'],
  [/\bVivaldi\//, 'Vivaldi'],
  [/\bYaBrowser\//, 'Yandex Browser'],
  [/\b(?:Firefox|FxiOS)\//, 'Firefox'],
  [/\b(?:Headless)?(?:Chrome|CriOS|Chromium)\//, 'Chrome'],
  [/\bVersion\/[\d.]+.*\bSafari\//, 'Safari'],
];

const systems: [RegExp, string][] = [
  [/\b(?:iPhone|iPad|iPod)\b/, 'iOS'],
  [/\bAndroid\b/, 'Android'],
  [/\bCrOS\b/, 'ChromeOS'],
  [/\bWindows\b/, 'Windows'],
  [/\bMac OS X\b|\bMacintosh\b/, 'macOS'],
  [/\bLinux\b/, 'Linux'],
];

const first = (ua: string, list: [RegExp, string][]) => list.find(([re]) => re.test(ua))?.[1] ?? '';

/**
 * Names the browser and system of a session in the words people use for
 * them. A tool such as curl that is not a browser keeps its own name.
 */
export function readAgent(ua: string): Agent {
  const browser = first(ua, browsers);
  const system = first(ua, systems);
  if (browser || system || ua.startsWith('Mozilla/')) return { browser, system };
  const tool = /^([A-Za-z][\w.-]{0,31})\//.exec(ua);
  return { browser: tool?.[1] ?? '', system: '' };
}
