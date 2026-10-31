-- The panel's three HTTPS modes become one domain enabled for Karvan: the
-- panel port always answers HTTPS with Karvan's own certificate, and a
-- handshake for the enabled domain's name gets that domain's certificate.
-- Domain mode keeps its domain (and a shared port) enabled. Self and off
-- enable none. Off, which answered plain HTTP, also takes plain HTTP for a
-- Karvan domain without a certificate; its plain requests are sent on to
-- HTTPS on the same port, so the old address still opens.
-- An installation set up before HTTPS existed has no http.tls row: off.
INSERT INTO settings (key, value)
SELECT 'http.tls_plain', 'true'
WHERE EXISTS (SELECT 1 FROM settings WHERE key = 'http.port')
  AND COALESCE((SELECT value FROM settings WHERE key = 'http.tls'), 'off') = 'off'
ON CONFLICT (key) DO UPDATE SET value = excluded.value;

UPDATE settings SET value = '0'
WHERE key = 'http.tls_domain'
  AND COALESCE((SELECT value FROM settings WHERE key = 'http.tls'), 'off') <> 'domain';

UPDATE settings SET value = 'false'
WHERE key = 'http.tls_shared'
  AND COALESCE((SELECT value FROM settings WHERE key = 'http.tls'), 'off') <> 'domain';

DELETE FROM settings WHERE key = 'http.tls';
