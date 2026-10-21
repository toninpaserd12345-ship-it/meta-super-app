#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UI_DIR="$ROOT_DIR/frontend"

# Use python3 for regex scanning to be completely independent of rg/grep differences
scan_count() {
  local pattern="$1"
  shift
  python3 -c "
import sys, re, os

pattern = re.compile(sys.argv[1])
count = 0
for path in sys.argv[2:]:
    if os.path.isfile(path) and path.endswith(('.vue', '.css')):
        with open(path, 'r', errors='ignore') as f:
            count += len(pattern.findall(f.read()))
    elif os.path.isdir(path):
        for root, _, files in os.walk(path):
            for file in files:
                if file.endswith(('.vue', '.css')):
                    with open(os.path.join(root, file), 'r', errors='ignore') as f:
                        count += len(pattern.findall(f.read()))
print(count)
" "$pattern" "$@"
}

echo "META-SUPPER-APP UI audit"
echo "routes=$(find "$UI_DIR/pages" -name '*.vue' | wc -l | tr -d ' ')"
echo "raw_colors_and_gradients=$(scan_count '(#[0-9A-Fa-f]{3,8}|rgba?\(|linear-gradient\(|radial-gradient\()' "$UI_DIR/pages" "$UI_DIR/components" "$UI_DIR/layouts")"
echo "gradients=$(scan_count '(linear-gradient\(|radial-gradient\()' "$UI_DIR/pages" "$UI_DIR/components" "$UI_DIR/layouts")"
inline_styles="$(scan_count 'style="|:style=' "$UI_DIR/pages" "$UI_DIR/components" "$UI_DIR/layouts")"
tiny_text="$(scan_count 'font-size:\s*(9|10|11)(\.5)?px' "$UI_DIR/assets" "$UI_DIR/pages" "$UI_DIR/components" "$UI_DIR/layouts")"
echo "inline_styles=$inline_styles"
echo "tiny_text=$tiny_text"

missing_alt="$(find "$UI_DIR/pages" "$UI_DIR/components" -name '*.vue' -print0 | xargs -0 perl -0777 -ne 'while (/<img\b[^>]*>/gs) { $count++ unless $& =~ /\balt\s*=/ } END { print $count || 0 }')"
icon_buttons_without_label="$(find "$UI_DIR/pages" "$UI_DIR/components" -name '*.vue' -print0 | xargs -0 perl -0777 -ne 'while (/<v-btn\b[^>]*\sicon="[^"]+"[^>]*>/gs) { $count++ unless $& =~ /(?:aria-label|:aria-label|title)=/ } END { print $count || 0 }')"

echo "images_missing_alt=$missing_alt"
echo "icon_buttons_missing_label=$icon_buttons_without_label"

if (( missing_alt > 0 || icon_buttons_without_label > 0 )); then
  echo "UI audit failed: fix accessibility errors above." >&2
  exit 1
fi

echo "UI audit passed. Raw colors, gradients, and inline styles are inventory metrics, not automatic failures."
