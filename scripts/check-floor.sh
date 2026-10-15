#!/usr/bin/env bash
# Floor rules from CONSTRAINTS.md that no linter covers. Exit 0 clean, 1 violation, 2 could not run.
set -euo pipefail

top=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "check-floor: not a git work tree" >&2; exit 2; }
cd "$top"

self="scripts/check-floor.sh"
mapfile -t files < <(git ls-files --cached --others --exclude-standard \
  | grep -vE "^(LICENSE|$self|CONSTRAINTS\.md)$" \
  | grep -vE '(^|/)pnpm-lock\.yaml$|\.(woff2?|png|jpg|ico)$' \
  | while read -r f; do [ -f "$f" ] && echo "$f"; done)

status=0
report() {
  local rule=$1 pattern=$2
  shift 2
  local hits
  hits=$(grep -nIE "$pattern" "$@" 2>/dev/null | cut -c1-160 || true)
  if [ -n "$hits" ]; then
    echo "[$rule]"
    echo "$hits" | sed 's/^/  /'
    status=1
  fi
}

[ ${#files[@]} -eq 0 ] && { echo "check-floor: clean"; exit 0; }

report silenced-checker '@ts-ignore|@ts-nocheck|eslint-disable|istanbul ignore|nosemgrep|gitleaks:allow' "${files[@]}"
report test-made-easier '\bt\.Skip(Now|f)?\(|\b(it|test|describe)\.(skip|only|todo)\(|\bx(it|describe)\(' "${files[@]}"
report unfinished-work '\b(TODO|FIXME|XXX)\b|[Nn]ot implemented' "${files[@]}"
report ai-reference '\b(Claude|Anthropic|ChatGPT|OpenAI|Copilot|Gemini|Kiro|LLM)\b|Co-authored-by|[Gg]enerated (by|with) (an )?AI' "${files[@]}"

mapfile -t web_src < <(printf '%s\n' "${files[@]}" | grep -E '^web/src/.*\.(ts|tsx|css)$' | grep -v '^web/src/styles/tokens\.css$' || true)
if [ ${#web_src[@]} -gt 0 ]; then
  report raw-color '#[0-9a-fA-F]{3,8}\b|\b(rgba?|hsla?|oklch|oklab)\(' "${web_src[@]}"
fi

if git rev-parse --verify -q HEAD >/dev/null; then
  msgs=$(git log --format='%h %an <%ae> %s%n%b' | grep -nE '\b(Claude|Anthropic|ChatGPT|OpenAI|Copilot|Gemini|Kiro|LLM)\b|Co-authored-by' || true)
  if [ -n "$msgs" ]; then echo "[ai-reference-in-history]"; echo "$msgs" | sed 's/^/  /'; status=1; fi
  authors=$(git log --format='%an <%ae>|%cn <%ce>' | sort -u | grep -v '^YrustPd <PdYrust@outlook.com>|YrustPd <PdYrust@outlook.com>$' || true)
  if [ -n "$authors" ]; then echo "[wrong-author]"; echo "$authors" | sed 's/^/  /'; status=1; fi
fi

[ $status -eq 0 ] && echo "check-floor: clean"
exit $status
