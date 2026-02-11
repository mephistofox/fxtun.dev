#!/bin/bash
# Sync cleaned copy of fxTunnel to public repo fxtun.dev
# Removes sensitive files from entire git history before pushing
set -e

PRIVATE_REPO="/home/fxcode/Проекты/fxTunnel"
WORK_DIR="/tmp/fxtun-dev-sync"
PUBLIC_REPO="mephistofox/fxtun.dev"
DESCRIPTION="fxTunnel — self-hosted open-source reverse tunnel server and client. Expose localhost to the internet with custom subdomains, TCP and UDP port forwarding. Fast ngrok/Cloudflare Tunnel alternative written in Go with Web UI, REST API, and cross-platform GUI client."

rm -rf "$WORK_DIR"
git clone --no-local "$PRIVATE_REPO" "$WORK_DIR"
cd "$WORK_DIR"

git filter-repo \
  --invert-paths \
  --path data/ \
  --path docs/plans/ \
  --path configs/nginx-fxtun-ru.conf \
  --path .github/workflows/deploy.yml \
  --path-glob 'web/public/*.txt' \
  --force

# Rewrite Go module path for public repo
OLD_MODULE="github.com/mephistofox/fxtunnel"
NEW_MODULE="github.com/mephistofox/fxtun.dev"
find . -name '*.go' -o -name 'go.mod' | xargs sed -i "s|${OLD_MODULE}|${NEW_MODULE}|g"
git add -A && git commit -m "chore: rewrite module path to ${NEW_MODULE}" --allow-empty

# Create repo if it doesn't exist
gh repo view "$PUBLIC_REPO" &>/dev/null || \
  gh repo create "$PUBLIC_REPO" --public --description "$DESCRIPTION"

git remote add origin "git@github.com:${PUBLIC_REPO}.git"
git push --force origin master
git push --force --tags origin 2>&1 | grep -v 'remote rejected' || true

rm -rf "$WORK_DIR"
echo "Done! https://github.com/${PUBLIC_REPO}"
