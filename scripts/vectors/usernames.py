#!/usr/bin/env python3
"""Golden username vectors from the panels' own rules.

The patterns are copied verbatim from Marzban 0.8.4 (app/models/user.py:16)
and PasarGuard 5.4.1 (app/models/validators.py:207), so Python's re decides
what each panel accepts. Run from the repository root:

    python3 scripts/vectors/usernames.py
"""

import json
import random
import re
import string

MARZBAN = re.compile(r"^(?=\w{3,32}\b)[a-zA-Z0-9-_@.]+(?:_[a-zA-Z0-9-_@.]+)*$")


def marzban_ok(name: str) -> bool:
    return bool(MARZBAN.match(name))


def pasarguard_ok(name: str) -> bool:
    if not (3 <= len(name) <= 128):
        return False
    if not re.match(r"^[a-zA-Z0-9-_@.]+$", name):
        return False
    return not re.search(r"[-_@.]{2,}", name)


def cases() -> list[str]:
    out = ["", "a", "ab", "abc", "ABC", "a1_", "___", "...", "abc.", ".abc", "-abc", "ab-c", "abc-d",
           "abc..d", "abc.-d", "abc_d", "abc__d", "a_b_c", "ab_c.d", "abc@x.y", "abc\n", "abc\nd",
           "abc d", " abc", "abc\t", "علی123", "ali\u00e9x", "ali\u0660\u0661\u0662", "abc:d", "abc?d", "abc[d",
           "abc\\d", "abc^d", "abc;d", "abc=d", "abc<d", "abc>d", "abc]d", "abc`d", "abc{d", "abc~d"]
    for n in (31, 32, 33, 34, 35, 127, 128, 129):
        out.append("a" * n)
        out.append("ab_" + "c" * n)
        out.append("abc." + "d" * n)
    for ch in string.printable:
        out.append("abc" + ch + "def")
        out.append(ch + "abcdef")
    rnd = random.Random(20261016)
    alphabet = string.ascii_letters + string.digits + "-_@.:\n é"
    for _ in range(3000):
        out.append("".join(rnd.choice(alphabet) for _ in range(rnd.randint(0, 40))))
    seen, uniq = set(), []
    for c in out:
        if c not in seen:
            seen.add(c)
            uniq.append(c)
    return uniq


def write(path: str, check) -> None:
    data = [{"name": c, "ok": check(c)} for c in cases()]
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=True, indent=0)
        f.write("\n")


write("internal/panel/marzban/testdata/usernames.json", marzban_ok)
write("internal/panel/pasarguard/testdata/usernames.json", pasarguard_ok)
