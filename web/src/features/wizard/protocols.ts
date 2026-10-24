const names: Readonly<Record<string, string>> = {
  vmess: 'VMess',
  vless: 'VLESS',
  trojan: 'Trojan',
  shadowsocks: 'Shadowsocks',
  wireguard: 'WireGuard',
  hysteria: 'Hysteria',
};

/** A protocol as panels write it ("VLESS"); one Karvan does not know keeps its code. */
export function protocolName(p: string): string {
  return names[p] ?? p;
}
