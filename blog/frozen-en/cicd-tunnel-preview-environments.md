---
title: "CI/CD with Tunnels: Preview Environments per PR"
date: 2026-03-05T14:00:00+03:00
draft: false
description: "Create a preview environment for every PR with fxTunnel and GitHub Actions: full workflow, Vercel comparison, and automatic cleanup."
tags: [ci-cd, preview, deployment, fxTunnel, github-actions, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "What is a preview environment for a pull request?"
    a: "A preview environment is a temporary environment that is automatically deployed when a PR is created, letting reviewers test changes via a public URL. fxTunnel creates a tunnel directly on the CI runner, making the application accessible from the internet without a dedicated server."
  - q: "How does tunnel preview differ from Vercel preview deployments?"
    a: "Vercel and Netlify only create previews for frontend apps (static sites, SSR). Tunnel preview works with any stack: a Go backend, a Python API, Java monolith, databases, queues — anything that runs on the CI runner is accessible via a public URL through fxTunnel."
  - q: "Is it safe to expose a CI runner through a tunnel?"
    a: "For preview environments with test data, yes. fxTunnel encrypts traffic via TLS. Use test databases and API keys, and do not attach production secrets. The tunnel closes automatically when the CI job finishes."
  - q: "Can I run E2E tests against a tunnel preview URL?"
    a: "Yes. Once the tunnel is created, you get a public HTTPS URL that you can pass to Playwright, Cypress, or any other E2E framework. This lets you test the application under conditions close to production."
  - q: "How much does tunnel preview cost compared to a staging server?"
    a: "fxTunnel is free for basic use. Even the paid plan at $5/mo is cheaper than a single staging server ($20-100/mo). And you get an isolated environment for every PR instead of a shared staging where changes from different developers collide."
---

## The Problem: Reviewing PRs Without a Live Preview Is Slow

You open a pull request diff on GitHub and... you see code. What you do not see is how the application actually behaves. To verify a PR manually, you have to switch branches locally, install dependencies, start the project, and click through scenarios by hand. That takes 10 to 15 minutes per PR, and most reviewers simply skip it.

The outcome is predictable: bugs slip into main, QA catches issues after the merge, and fixes turn into yet more pull requests. The feedback cycle stretches from minutes to hours or days. A [local tunnel](/blog/what-is-tunneling/) combined with CI/CD can fix this: every PR gets its own public URL with a running application.

## Traditional Approaches to Preview Environments

Preview environments are not a new idea. Vercel and Netlify have been creating preview deployments for every PR for years. But these solutions share a serious limitation: they only work for the frontend.

### Vercel / Netlify: Frontend Only

Vercel deploys previews for Next.js, Astro, and other frameworks with SSR or static generation. Netlify does the same for static sites and serverless functions. Both services are excellent for frontend projects.

But if your application is a Go backend with PostgreSQL, a Python API with Celery and Redis, or a Java monolith, Vercel will not help. You need a preview for the full stack, not just the frontend.

### Staging Server: One for Everyone

The classic staging server is a shared environment where changes are deployed for testing. The problem is that there is one staging but many PRs. Two developers deploy simultaneously, and one set of changes overwrites the other. The staging queue slows down the entire team.

### Kubernetes Per-PR Namespaces: Powerful but Complex

Some teams create a separate Kubernetes namespace for every PR. This works, but it requires serious infrastructure: Helm charts, Ingress controllers, wildcard DNS, automated cleanup. For a team of 5 to 10 people, it is overkill.

## Full-Stack Preview with Tunnels: The Idea

The approach is straightforward. The CI runner builds and starts your application, fxTunnel creates a [public URL](/blog/expose-localhost-to-internet/) pointing to it, and a bot comments on the PR with the link. The reviewer clicks the link and sees a working application.

```
PR opened
    |
    v
CI runner (GitHub Actions)
    |
    |-- 1. Build the app (docker compose up)
    |-- 2. Start fxTunnel -> get a public URL
    |-- 3. Comment on the PR with the preview link
    |-- 4. Wait (or run E2E tests)
    |
PR closed / merged
    |
    +-- 5. Tunnel closes automatically
```

The benefits are clear: the preview works with any stack, no separate infrastructure is needed, and the tunnel shuts down automatically when the CI job finishes.

## GitHub Actions: Full Workflow

Below is a ready-to-use workflow for GitHub Actions. It builds the application with Docker Compose, starts fxTunnel, and comments on the PR with the public URL. You can copy it and adapt it to your project.

### Main Workflow

```yaml
# .github/workflows/preview.yml
name: PR Preview Environment

on:
  pull_request:
    types: [opened, synchronize, reopened]

concurrency:
  group: preview-${{ github.event.pull_request.number }}
  cancel-in-progress: true

jobs:
  preview:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Start application
        run: |
          docker compose -f docker-compose.preview.yml up -d --build
          # Wait until the app is reachable
          for i in $(seq 1 30); do
            curl -sf http://localhost:8080/health && break
            echo "Waiting for app to start... ($i/30)"
            sleep 2
          done

      - name: Install fxTunnel
        run: curl -fsSL https://fxtun.ru/install.sh | bash

      - name: Start tunnel and get URL
        id: tunnel
        run: |
          # Start the tunnel in the background and capture the URL
          fxtunnel http 8080 --log-file /tmp/tunnel.log &
          TUNNEL_PID=$!
          echo "tunnel_pid=$TUNNEL_PID" >> "$GITHUB_OUTPUT"

          # Wait for the URL to appear in the logs
          for i in $(seq 1 15); do
            TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' /tmp/tunnel.log || true)
            if [ -n "$TUNNEL_URL" ]; then
              echo "preview_url=$TUNNEL_URL" >> "$GITHUB_OUTPUT"
              echo "Preview URL: $TUNNEL_URL"
              break
            fi
            sleep 1
          done

          if [ -z "$TUNNEL_URL" ]; then
            echo "Failed to get tunnel URL"
            exit 1
          fi

      - name: Comment PR with preview URL
        uses: actions/github-script@v7
        with:
          script: |
            const previewUrl = '${{ steps.tunnel.outputs.preview_url }}';
            const body = `## Preview Environment

            | | |
            |---|---|
            | **URL** | ${previewUrl} |
            | **Commit** | \`${context.sha.substring(0, 7)}\` |
            | **Status** | Active |

            This preview will be available while the CI job is running.`;

            // Delete the old comment if it exists
            const comments = await github.rest.issues.listComments({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
            });
            const botComment = comments.data.find(c =>
              c.body.includes('## Preview Environment')
            );
            if (botComment) {
              await github.rest.issues.deleteComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                comment_id: botComment.id,
              });
            }

            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              body: body,
            });

      - name: Run E2E tests against preview
        run: |
          npx playwright test --base-url=${{ steps.tunnel.outputs.preview_url }}

      - name: Keep preview alive for manual testing
        if: success()
        run: |
          echo "Preview is live at: ${{ steps.tunnel.outputs.preview_url }}"
          echo "Keeping alive for 20 minutes for manual review..."
          sleep 1200

      - name: Cleanup
        if: always()
        run: |
          kill ${{ steps.tunnel.outputs.tunnel_pid }} || true
          docker compose -f docker-compose.preview.yml down
```

### Docker Compose for Preview

```yaml
# docker-compose.preview.yml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/preview
      - REDIS_URL=redis://redis:6379
      - APP_ENV=preview
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: preview
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user"]
      interval: 2s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
```

For more on using Docker with tunnels, see [Docker + tunnel](/blog/docker-tunnel-expose-containers/).

## GitLab CI: Brief Example

Here is the equivalent workflow for GitLab CI. The principle is the same: start the application, create a tunnel, comment on the merge request.

```yaml
# .gitlab-ci.yml
preview:
  stage: test
  image: docker:24
  services:
    - docker:24-dind
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
    - docker compose -f docker-compose.preview.yml up -d --build
    - curl -fsSL https://fxtun.ru/install.sh | bash
    - fxtunnel http 8080 --log-file /tmp/tunnel.log &
    - sleep 5
    - TUNNEL_URL=$(grep -oP 'https://[a-z0-9-]+\.fxtun\.dev' /tmp/tunnel.log)
    - |
      curl --request POST \
        --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" \
        --header "Content-Type: application/json" \
        --data "{\"body\": \"Preview: ${TUNNEL_URL}\"}" \
        "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/merge_requests/${CI_MERGE_REQUEST_IID}/notes"
    - echo "Preview live at ${TUNNEL_URL}"
    - sleep 1200
  after_script:
    - docker compose -f docker-compose.preview.yml down
```

## What Tunnel Preview Gives Your Team

What changes when every PR comes with a live link? Quite a lot, actually.

### Test Webhooks Directly from a PR

If your application receives [webhooks from Stripe, GitHub, or Telegram](/blog/webhook-testing-with-tunnel/), the preview environment lets you point a test webhook at the PR URL and verify the integration before merging. A QA engineer can trigger a test payment in Stripe and see how the application handles it.

### Share with QA and Designers

The link in the PR comment lets anyone on the team open the preview in a browser without cloning the repository, installing dependencies, or starting the project locally. A designer checks the UI, a product manager verifies business logic, and QA explores edge cases.

### Run E2E Tests Against a Real URL

The public HTTPS URL from fxTunnel can be passed to Playwright, Cypress, or Selenium. Tests run against an application that operates under near-production conditions: real HTTP, TLS, DNS resolution. This is more reliable than testing against `localhost`.

### Test Mobile Integrations

If a mobile app connects to the backend, QA can enter the PR URL into a test build configuration and verify the integration on a real device.

## Cost Comparison: Tunnel Preview vs Alternatives

| Approach | Cost | Full stack | PR isolation | Setup time |
|---|---|---|---|---|
| **fxTunnel (free)** | $0 | Yes | Yes | 10 minutes |
| **fxTunnel (paid)** | from $5/mo | Yes | Yes | 10 minutes |
| **Vercel preview** | $0-20/mo | Frontend only | Yes | 5 minutes |
| **Netlify preview** | $0-19/mo | Frontend only | Yes | 5 minutes |
| **Staging server** | $20-100/mo | Yes | No (shared) | Hours |
| **K8s per-PR namespaces** | $50-200/mo | Yes | Yes | Days |

For full-stack projects, fxTunnel gives you an isolated preview for every PR with support for any stack at minimal cost. Vercel and Netlify remain a strong choice for purely frontend projects. You can also see how fxTunnel stacks up against [ngrok and Cloudflare Tunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

## Cleanup: Closing the Tunnel When a PR Is Merged or Closed

A tunnel started inside a CI job closes automatically when the job finishes. But if you use `sleep` to keep the preview alive, you need a mechanism to cancel it when the PR is merged or closed.

GitHub Actions handles this through `concurrency` and a separate workflow that fires on PR close.

### Cleanup Workflow on PR Close

```yaml
# .github/workflows/preview-cleanup.yml
name: Cleanup Preview

on:
  pull_request:
    types: [closed]

jobs:
  cleanup:
    runs-on: ubuntu-latest
    steps:
      - name: Cancel preview workflow
        uses: actions/github-script@v7
        with:
          script: |
            // Find active workflow runs for this PR
            const runs = await github.rest.actions.listWorkflowRuns({
              owner: context.repo.owner,
              repo: context.repo.repo,
              workflow_id: 'preview.yml',
              status: 'in_progress',
            });

            for (const run of runs.data.workflow_runs) {
              // Check that the run belongs to our PR
              if (run.pull_requests.some(pr => pr.number === context.issue.number)) {
                await github.rest.actions.cancelWorkflowRun({
                  owner: context.repo.owner,
                  repo: context.repo.repo,
                  run_id: run.id,
                });
                console.log(`Cancelled workflow run ${run.id}`);
              }
            }

      - name: Update PR comment
        uses: actions/github-script@v7
        with:
          script: |
            const comments = await github.rest.issues.listComments({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
            });
            const botComment = comments.data.find(c =>
              c.body.includes('## Preview Environment')
            );
            if (botComment) {
              await github.rest.issues.updateComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                comment_id: botComment.id,
                body: `## Preview Environment\n\n**Status:** Closed (PR ${context.payload.action})`,
              });
            }
```

The key detail is the `concurrency` section in the main workflow. The `cancel-in-progress: true` parameter ensures that when a new commit is pushed to the PR, the old preview job is cancelled and a new one starts with updated code. When the PR is closed, the separate cleanup workflow cancels the active preview.

## Tips for Production-Ready Setup

### Cache Docker Layers

Building Docker images on every push can be slow. Use layer caching to speed things up:

```yaml
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Cache Docker layers
        uses: actions/cache@v4
        with:
          path: /tmp/.buildx-cache
          key: ${{ runner.os }}-buildx-${{ github.sha }}
          restore-keys: ${{ runner.os }}-buildx-
```

### Health Check Before Opening the Tunnel

Do not start the tunnel until the application is ready. Use a health check endpoint:

```yaml
      - name: Wait for application
        run: |
          for i in $(seq 1 30); do
            if curl -sf http://localhost:8080/health; then
              echo "Application is ready"
              exit 0
            fi
            echo "Waiting... ($i/30)"
            sleep 2
          done
          echo "Application failed to start"
          exit 1
```

### Limit Preview Lifetime

Set a reasonable `timeout-minutes` for the job and a sensible `sleep` duration. 20 to 30 minutes is usually enough for manual testing. For automated E2E tests, the preview only needs to stay alive for the duration of the test run.

## FAQ

### What is a preview environment for a pull request?

Think of it as a disposable deployment that spins up automatically when someone opens a PR. Reviewers get a public URL they can click to see the running application. With fxTunnel, a [tunnel](/blog/what-is-tunneling/) starts directly on the CI runner, so no dedicated server is needed.

### How does tunnel preview differ from Vercel preview deployments?

Vercel and Netlify target the frontend -- static sites and SSR apps. A tunnel-based preview works with any stack: Go backends, Python APIs, Java monoliths, databases, queues. If it runs on the CI runner, it is accessible through the tunnel's public URL.

### Is it safe to expose a CI runner through a tunnel?

For preview environments running test data, yes. fxTunnel encrypts all traffic via TLS. Stick to test databases and test API keys, and keep production secrets out of the workflow. The tunnel shuts down automatically when the CI job ends.

### Can I run E2E tests against a tunnel preview URL?

Absolutely. The tunnel gives you a public HTTPS URL that you can hand to Playwright, Cypress, or any other E2E framework. Your tests run against an application behind real HTTP, TLS, and DNS -- much closer to production than localhost.

### How much does tunnel preview cost compared to a staging server?

fxTunnel is free for basic use, and even the paid plan at $5/mo costs far less than a single staging server ($20-100/mo). The real win is isolation: every PR gets its own environment instead of a shared staging where developers step on each other's changes.
