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

# Create repo if it doesn't exist
gh repo view "$PUBLIC_REPO" &>/dev/null || \
  gh repo create "$PUBLIC_REPO" --public --description "$DESCRIPTION"

git remote add origin "git@github.com:${PUBLIC_REPO}.git"
git push --force --tags origin master

rm -rf "$WORK_DIR"
echo "Done! https://github.com/${PUBLIC_REPO}"
