#!/usr/bin/env python3
"""Golden subscription tokens from the panels' own algorithms.

Copied from Marzban 0.8.4 app/utils/jwt.py (create_subscription_token)
and PasarGuard 5.4.1 app/utils/jwt.py (create_subscription_token, and the
legacy branches of get_subscription_payload), with the clock fixed.
Run from the repository root:

    python3 scripts/vectors/subtokens.py
"""

import hmac
import json
from base64 import b64encode
from hashlib import sha256

SECRET = "karvan-test-secret-0123456789abcdef0123456789abcdef0123456789ab"


def marzban(username: str, ts: int) -> str:
    data = username + "," + str(ts)
    b64 = b64encode(data.encode(), altchars=b"-_").decode().rstrip("=")
    sign = b64encode(sha256((b64 + SECRET).encode()).digest(), altchars=b"-_").decode()[:10]
    return b64 + sign


def pasarguard(user_id: int, ts: int) -> str:
    data = "v3," + str(user_id) + "," + str(ts)
    b64 = b64encode(data.encode(), altchars=b"-_").decode().rstrip("=")
    sig = b64encode(hmac.new(SECRET.encode(), b64.encode(), sha256).digest(), altchars=b"-_").decode().rstrip("=")
    return b64 + "." + sig


def legacy_hex(payload: str) -> str:
    b64 = b64encode(payload.encode(), altchars=b"-_").decode().rstrip("=")
    return b64 + sha256((b64 + SECRET).encode()).hexdigest()[:10]


vectors = {
    "secret": SECRET,
    "marzban": [{"username": u, "ts": ts, "token": marzban(u, ts)} for u, ts in
                [("ali", 1785654065), ("reza.k@shop", 1700000001), ("a_b-c", 1)]],
    "pasarguard": [{"id": i, "ts": ts, "token": pasarguard(i, ts)} for i, ts in
                   [(97318, 1785654065), (1, 1700000000), (123456789, 2000000000)]],
    "legacyHex": [{"payload": p, "token": legacy_hex(p)} for p in ["v3,97318,1785654065", "ali,1785654065"]],
}
with open("internal/panel/subtoken/testdata/vectors.json", "w", encoding="utf-8") as f:
    json.dump(vectors, f, indent=1)
    f.write("\n")
