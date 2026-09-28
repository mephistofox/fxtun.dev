#!/usr/bin/env bash
# Visual acceptance: build the blog for localhost and screenshot key pages
# at desktop and phone width. Usage: blog/scripts/shots.sh <out-dir>
set -euo pipefail
OUT=${1:?out dir}; mkdir -p "$OUT"
cd "$(dirname "$0")/.."
hugo --quiet -D --baseURL http://localhost:8791/ -d public-local
python3 -m http.server -d public-local 8791 >/dev/null 2>&1 & PID=$!
trap 'kill $PID' EXIT; sleep 1
for p in "" "rubric/guides/" "tcp-udp-tunneling-explained/" "diary/10-retrospective.html" "styleguide/"; do
  n=$(echo "${p:-home}" | tr '/.' '__')
  google-chrome --headless=new --hide-scrollbars --screenshot="$OUT/$n-1440.png" --window-size=1440,4000 --virtual-time-budget=4000 "http://localhost:8791/$p" 2>/dev/null
  google-chrome --headless=new --hide-scrollbars --screenshot="$OUT/$n-390.png" --window-size=390,7000 --virtual-time-budget=4000 "http://localhost:8791/$p" 2>/dev/null
  # Same-origin probe: load the page in a 390px frame and print its scrollWidth.
  printf '<iframe id=f src="/%s" width=390 height=900></iframe><pre id=o>?</pre><script>f.onload=function(){var d=f.contentDocument.documentElement;o.textContent=d.scrollWidth+" "+d.clientWidth}</script>' "$p" > public-local/_probe.html
  w=$(google-chrome --headless=new --virtual-time-budget=5000 --dump-dom http://localhost:8791/_probe.html 2>/dev/null | grep -o '<pre id="o">[^<]*' | cut -d'>' -f2)
  echo "scrollWidth clientWidth ${p:-/} @390: $w"
done
ls "$OUT"
