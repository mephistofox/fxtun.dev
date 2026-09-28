---
title: "Remote Pair Programming via Tunnel — Instant Setup"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Use tunnels for remote pair programming: share localhost with teammates, VS Code Live Share + tunnel, and collaborative debugging workflows."
tags: [pair-programming, remote-development, collaboration, fxTunnel, debugging, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "How do I share my local dev environment with a remote teammate?"
    a: "Run fxtunnel http 3000 to create a public HTTPS URL for your localhost. Send the link to your teammate — they open it in a browser and see your running application in real time. No deployment, no VPN, no firewall configuration required."
  - q: "Can I use a tunnel together with VS Code Live Share?"
    a: "Yes. VS Code Live Share handles collaborative code editing, while fxTunnel provides a public URL so both developers can see the running application in a browser. Start Live Share for the code and fxtunnel http 3000 for the app — the combination covers both editing and testing."
  - q: "Is it safe to share my dev environment through a tunnel?"
    a: "Yes, with basic precautions. fxTunnel encrypts all traffic with TLS. Use test data, do not expose admin panels, keep secrets in environment variables, and close the tunnel when the session ends. For extra security, add basic HTTP authentication."
  - q: "Can two developers connect to the same tunnel URL?"
    a: "Yes. The tunnel URL is a regular HTTPS address — any number of people can open it simultaneously. Each person sees the same application state. This is useful for code review sessions, QA testing, and client demos during pair programming."
  - q: "Does the tunnel add noticeable latency for pair programming?"
    a: "Typically no. fxTunnel adds 10-30ms of latency, which is imperceptible for web applications. The experience is comparable to browsing any website. For real-time collaborative editing, use VS Code Live Share (which has its own low-latency protocol) alongside the tunnel."
---

## The Problem: Remote Collaboration Without Access to Each Other's Machines

Remote pair programming sounds simple in theory: two developers work on the same code at the same time. In practice, the logistics are painful. Developer A runs a web app on `localhost:3000`. Developer B needs to see it, test it, and debug it. But `localhost` is just that — local. It does not exist outside the machine.

The usual workarounds are clumsy:

- **Screen sharing** — Developer B watches a video stream. They cannot interact with the app, open DevTools, or test on their own device. Latency makes it frustrating.
- **Deploy to staging** — every change requires a push, a build, and a deploy. The feedback cycle stretches from seconds to minutes.
- **VPN** — connecting two developers via a corporate VPN requires infrastructure, configuration, and IT support. Overkill for a pair programming session.
- **Clone and run locally** — Developer B pulls the branch, installs dependencies, configures environment variables, and starts the project. This can take 15 minutes or more, and environment differences cause "works on my machine" issues.

None of these approaches give Developer B instant, interactive access to Developer A's running application. A [tunnel](/blog/what-is-tunneling/) does.

## The Solution: A Tunnel Turns Localhost Into a Shared Environment

A tunnel creates a public HTTPS URL that points to your local server. When your teammate opens the link, they see your running application — the same state, the same data, the same behavior. Every code change you make is reflected immediately (if you are using a dev server with hot reload).

The workflow is straightforward:

1. Developer A starts the application locally.
2. Developer A runs `fxtunnel http 3000`.
3. Developer A sends the public URL to Developer B.
4. Developer B opens the URL in their browser.
5. Both developers see the same live application.

```bash
# Developer A: start the app and create a tunnel
npm run dev
# -> http://localhost:3000

fxtunnel http 3000
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://pair-abc.fxtun.dev
Forwarding:  https://pair-abc.fxtun.dev -> http://localhost:3000

Press Ctrl+C to stop
```

Developer B opens `https://pair-abc.fxtun.dev` in their browser — they see the app, can click through it, open DevTools, inspect network requests, and test functionality independently. No setup on their side.

If you are curious about how tunneling works under the hood, [What Is Tunneling](/blog/what-is-tunneling/) covers the mechanics.

## Scenario 1: Code Review on a Live Application

Code review in a GitHub diff is useful but limited. You can see what changed in the code, but you cannot see how the application actually behaves. Does the new button align correctly? Does the form validation fire on the right inputs? Does the API return the expected response?

A tunnel lets the reviewer interact with the running application during the review.

### The Workflow

```bash
# The author of the PR starts the app on their branch
git checkout feature/new-checkout-flow
npm run dev
# -> http://localhost:3000

# Open a tunnel
fxtunnel http 3000
# -> https://review-abc.fxtun.dev
```

The author posts the URL in the PR comment:

```markdown
## Preview

Live preview: https://review-abc.fxtun.dev

Test the new checkout flow:
1. Add an item to the cart
2. Click "Checkout"
3. Fill in the form and submit
```

The reviewer opens the link, tests the flow, and leaves comments directly in the PR — informed by hands-on experience, not just reading code.

### Frontend + Backend Review

If the PR affects both the frontend and the API, open two tunnels:

```bash
# Terminal 1 — frontend
fxtunnel http 3000
# -> https://front-review.fxtun.dev

# Terminal 2 — API
fxtunnel http 8000
# -> https://api-review.fxtun.dev
```

Update the frontend config to point to the API tunnel:

```bash
# .env.local
REACT_APP_API_URL=https://api-review.fxtun.dev
```

The reviewer can test the entire stack without pulling the branch or running anything locally.

## Scenario 2: VS Code Live Share + Tunnel

VS Code Live Share is excellent for collaborative editing: two developers see the same code, can edit simultaneously, and share terminals. But Live Share does not give the guest access to the running application in a browser. The guest sees the code but cannot open `localhost:3000` — it only exists on the host's machine.

A tunnel fills this gap.

### Setup

**Host (Developer A):**

1. Open VS Code and start a Live Share session (`Ctrl+Shift+P` -> `Live Share: Start Collaboration Session`).
2. Start the dev server in the VS Code terminal.
3. Open a tunnel to the dev server port.

```bash
# In the VS Code terminal
npm run dev
# -> http://localhost:3000

fxtunnel http 3000
# -> https://liveshare-abc.fxtun.dev
```

4. Share the Live Share link and the tunnel URL with the guest.

**Guest (Developer B):**

1. Join the Live Share session — they can now edit code alongside the host.
2. Open the tunnel URL in their browser — they can see the running app.
3. Every code change from either developer is reflected in the app (via hot reload).

### VS Code Settings for Tunnel Integration

You can create a VS Code task to start the tunnel automatically alongside the dev server:

```json
// .vscode/tasks.json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Dev Server",
      "type": "shell",
      "command": "npm run dev",
      "isBackground": true,
      "problemMatcher": []
    },
    {
      "label": "Tunnel",
      "type": "shell",
      "command": "fxtunnel http 3000",
      "isBackground": true,
      "problemMatcher": [],
      "dependsOn": "Dev Server"
    },
    {
      "label": "Pair Programming",
      "dependsOn": ["Dev Server", "Tunnel"],
      "problemMatcher": []
    }
  ]
}
```

Run the "Pair Programming" task, and both the dev server and the tunnel start together. The tunnel URL appears in the terminal output — copy and share it.

## Scenario 3: Collaborative Debugging

Debugging is harder remotely. When a bug surfaces, you need your teammate to see the exact state of the application — the broken page, the failing request, the error in the console. Screen sharing shows a compressed video. A tunnel gives full interactive access.

### Debugging a Frontend Bug

Developer B reports a visual bug. Developer A opens a tunnel and asks Developer B to reproduce the bug on the live URL.

```bash
# Developer A
fxtunnel http 3000
# -> https://debug-abc.fxtun.dev
```

Developer B opens the URL, reproduces the bug, and opens DevTools. They can:

- Inspect the DOM and CSS to identify layout issues.
- Check the console for JavaScript errors.
- Examine network requests for failed API calls.
- Test on their own device and browser (which may be the one where the bug occurs).

This is more productive than asking Developer B to describe the bug over chat or record a video.

### Debugging an API with Inspector

When debugging API issues, the built-in [Traffic Inspector](/blog/traffic-inspector-debug-requests/) is invaluable. Inspector shows every HTTP request passing through the tunnel in real time — headers, request body, response status, and response body.

```bash
# Start a tunnel with Inspector enabled (from $5/mo)
fxtunnel http 8000
```

Both developers can view the Inspector in their browser. When Developer B makes a request through the tunnel, Developer A sees the exact payload and response. If the request fails, Developer A can use the Replay feature to resend it while stepping through the code in a debugger.

### Debugging with Multiple Services

Complex applications involve multiple services. During a pair programming session, you can expose all of them:

```bash
# Terminal 1 — frontend (React on port 3000)
fxtunnel http 3000
# -> https://front-debug.fxtun.dev

# Terminal 2 — API (Express on port 8000)
fxtunnel http 8000
# -> https://api-debug.fxtun.dev

# Terminal 3 — WebSocket server (port 8081)
fxtunnel http 8081
# -> https://ws-debug.fxtun.dev
```

Your teammate can interact with the frontend, hit the API directly, and even connect to the WebSocket server — all through public URLs. This level of access makes remote debugging as effective as sitting at the same desk.

## Scenario 4: Live QA Testing During Development

QA engineers often cannot test a feature until it reaches the staging server. This creates a bottleneck: the developer finishes the feature, waits for the deploy, and then waits for QA feedback. With a tunnel, QA can test the feature while the developer is still working on it.

### The Workflow

```bash
# Developer opens a tunnel and shares the URL with QA
fxtunnel http 3000
# -> https://qa-test.fxtun.dev
```

The developer posts the URL in the team channel:

```
Feature "user profile editing" is ready for testing:
https://qa-test.fxtun.dev/profile

Test scenarios:
- Edit name and save
- Upload avatar (JPEG, PNG)
- Change password
```

The QA engineer opens the link and tests. When they find a bug:

1. QA reports the bug with steps to reproduce.
2. The developer fixes it locally.
3. The fix is instantly available through the tunnel (hot reload).
4. QA refreshes and verifies the fix.

The cycle from "bug found" to "fix verified" takes minutes instead of hours.

### Mobile QA Testing

If the feature has a mobile aspect, QA can open the tunnel URL on a real phone. No special setup -- it is a regular HTTPS link. The [mobile testing article](/blog/mobile-app-testing-tunnel/) covers more scenarios.

## Scenario 5: Client Demos During Pair Sessions

Sometimes pair programming involves a non-technical stakeholder — a product manager, a designer, or a client. They need to see the application, not the code. A tunnel URL lets them follow along in real time.

```bash
# Developer shares the tunnel URL during a call
fxtunnel http 3000
# -> https://demo-pair.fxtun.dev
```

The client opens the URL on their device and sees the application. The developer makes changes, and the client sees them after a page refresh. This is more engaging than screen sharing because the client can navigate independently, test on their own device, and experience the app at their own pace.

For professional client-facing demos, consider using a custom domain (from $5/mo):

```bash
fxtunnel http 3000 --domain preview.yourcompany.com
```

A branded URL looks more polished than a random subdomain -- and the [demo article](/blog/demo-localhost-without-deploy/) has more tips on making these sessions smooth.

## Setting Up a Pair Programming Environment

Here is a complete setup checklist for a productive remote pair programming session with tunnels.

### Prerequisites

```bash
# Install fxTunnel
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify installation
fxtunnel --version
```

### Configuration File for Repeated Sessions

If you pair program regularly, create a configuration to avoid typing tunnel commands every time:

```bash
# Start multiple tunnels for a full-stack project
#!/bin/bash
# pair-session.sh

echo "Starting pair programming environment..."

# Start the frontend
npm run dev &
FRONTEND_PID=$!

# Start the API
cd api && npm start &
API_PID=$!

# Wait for servers to start
sleep 3

# Open tunnels
fxtunnel http 3000 --log-file /tmp/tunnel-front.log &
fxtunnel http 8000 --log-file /tmp/tunnel-api.log &

echo "Frontend tunnel: check /tmp/tunnel-front.log for URL"
echo "API tunnel: check /tmp/tunnel-api.log for URL"
echo "Press Ctrl+C to stop all"

wait
```

### Environment Variables for Tunnel URLs

Configure your application to use environment variables for API URLs, so switching between localhost and tunnel takes a single line:

```bash
# .env.development
API_URL=http://localhost:8000

# .env.tunnel (for pair programming sessions)
API_URL=https://api-pair.fxtun.dev
```

```bash
# Start with tunnel configuration
DOTENV=.env.tunnel npm run dev
```

## Security During Pair Programming Sessions

Opening your local server to the internet requires care. Here are the ground rules (the [safe localhost exposure guide](/blog/expose-localhost-to-internet/) goes deeper if you want the full picture).

### Best Practices

- **Close the tunnel when the session ends.** Hit `Ctrl+C`. Do not leave tunnels running overnight.
- **Use test data.** Never expose a development database with real user data through a tunnel.
- **Keep secrets in `.env` files.** API keys, database passwords, and tokens should never be in source code.
- **Do not expose admin interfaces.** If your app has an `/admin` route, make sure it requires authentication.

### What Not to Do

| Bad Practice | Risk | Better Approach |
|---|---|---|
| Tunnel running overnight | Anyone with the URL has access | Close with `Ctrl+C` after the session |
| Real database exposed | User data leak | Use seed or test data |
| Hardcoded API keys | Keys visible through Inspector | Use `.env` + `.gitignore` |
| Open `/admin` panel | Unauthorized access | Require authentication |

### Access Control

For sessions that last longer than a quick pair programming call, consider additional security measures:

```bash
# Use a custom domain for consistent, recognizable URLs (from $5/mo)
fxtunnel http 3000 --domain pair.yourteam.dev
```

Adding basic HTTP authentication at the application level restricts access to invited team members only.

## Comparison: Tunnel vs Other Remote Collaboration Tools

| Method | Interactive App Access | Code Editing | Setup Time | Cost |
|---|---|---|---|---|
| **fxTunnel** | Full browser access | No (combine with editor) | 30 seconds | Free (custom domain from $5/mo) |
| **VS Code Live Share** | No (code only) | Yes | 2 minutes | Free |
| **fxTunnel + Live Share** | Full browser access | Yes | 3 minutes | Free |
| **Screen sharing (Zoom)** | Watch only | No | 0 minutes | Free (with limits) |
| **Deploy to staging** | Full access | No | 15-60 minutes | $5-50/mo |
| **VPN** | Full network access | No | Hours (initial setup) | $10-50/mo |

Pairing fxTunnel with VS Code Live Share covers both bases: collaborative code editing and full interactive access to the running application.

## Pricing for Pair Programming Teams

For most pair programming scenarios, the free tier is all you need -- unlimited tunnels, HTTPS, and stable URLs. If your workflow involves regular API debugging or client-facing demos with a branded domain, the paid plans (from $5/mo for custom domains and Inspector, from $10/mo for 10+ tunnels) fill those gaps.

Wondering how fxTunnel compares to alternatives? The [ngrok vs Cloudflare Tunnel vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/) article breaks that down.

## FAQ

### How do I share my local dev environment with a remote teammate?

One command -- `fxtunnel http 3000` -- gives you a public HTTPS URL. Send that link to your teammate and they are looking at your running application in their own browser, with full DevTools access. No deploy step, no VPN setup, no firewall rules to tweak.

### Can I use a tunnel together with VS Code Live Share?

Absolutely. Live Share handles the code editing side, and fxTunnel gives both of you a URL to the running app. Start Live Share for the codebase, run `fxtunnel http 3000` for the application -- together they cover editing and testing without gaps.

### Is it safe to share my dev environment through a tunnel?

It is, as long as you follow common sense. All traffic goes through TLS. Stick to test data, avoid exposing admin routes, keep secrets in environment variables, and close the tunnel once the session wraps up. Adding basic HTTP auth gives you one more layer if you want it.

### Can two developers connect to the same tunnel URL?

The URL is just a regular HTTPS address, so there is no limit on how many people can open it at once. Everyone sees the same live application state, which makes it great for code reviews, QA walkthroughs, and client demos.

### Does the tunnel add noticeable latency for pair programming?

In practice, no. The tunnel adds roughly 10-30 ms, which feels no different from visiting any other website. For the real-time code editing part, VS Code Live Share uses its own low-latency protocol separately from the tunnel, so neither side feels sluggish.
