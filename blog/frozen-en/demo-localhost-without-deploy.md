---
title: "How to Demo Your Local Project to Clients Without Deploying"
date: 2026-02-25T14:00:00+03:00
draft: false
description: "Share localhost with clients without deploying: fxTunnel guide for React, Vue, Next.js, Express, and FastAPI. Public HTTPS URL in 30 seconds."
tags: [demo, localhost, sharing, fxTunnel, freelance, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "How do I show my local site to a client without deploying?"
    a: "Run fxTunnel with the command fxtunnel http 3000 — in seconds you get a public HTTPS address like https://abc123.fxtun.dev that you can send to your client. No deployment, no servers, no DNS configuration required."
  - q: "Is it safe to share localhost through a tunnel?"
    a: "Yes, as long as you follow basic rules. fxTunnel encrypts traffic with TLS. Use test data, close the tunnel after the demo (Ctrl+C), and do not leave it running overnight. For extra protection, restrict access by IP or add basic authentication."
  - q: "Can I demo frontend and backend at the same time?"
    a: "Yes. Open two tunnels: fxtunnel http 3000 for the frontend and fxtunnel http 8000 for the API. Or configure a proxy in your framework (e.g., proxy in package.json for React) so API requests go through a single tunnel."
  - q: "Does the URL change when I restart the tunnel?"
    a: "In fxTunnel SaaS, the URL stays the same even on the free tier. With a custom domain (from $5/mo), you have full control over the address — for example, demo.yoursite.com. Your client does not need to update bookmarks."
  - q: "Can I use a tunnel to demo a mobile app?"
    a: "Yes. If your mobile app talks to a local API, open a tunnel to the API server port. Replace localhost with the tunnel's public URL in your app config — and test on a real device without being tied to the same Wi-Fi network."
---

## The Problem: Deploying Just for a Demo

You are working on a project, the client wants to see progress, and you need to show results fast. Everything works on your machine, but the client cannot open your `localhost:3000`. So the familiar cycle begins — set up CI/CD, wait for the build, configure the staging server, fix environment variables, double-check that nothing broke during deployment. Anywhere from 20 minutes to several hours, gone.

For freelancers, it is even worse: a dedicated staging server means extra costs that are not always justified. And deploying to a free hosting platform means spending time adapting configuration and dealing with platform limitations.

All of this for a single demo that lasts 15 minutes.

## The Solution: A Tunnel Gives You a Public URL in 30 Seconds

A [localhost tunnel](/blog/what-is-tunneling/) makes your local server accessible at a public HTTPS address with a single command. No deployment, no CI/CD, no staging servers. fxTunnel creates an encrypted connection between your computer and a public server that issues a URL like `https://abc123.fxtun.dev`. The client opens the link in a browser and sees exactly what is on your screen.

How it works:

1. You run `fxtunnel http 3000` in your terminal.
2. fxTunnel establishes an encrypted connection to a public server.
3. The server issues a public HTTPS address.
4. All requests to that address are forwarded to your `localhost:3000`.

## Step-by-Step: Demo with fxTunnel

### Installation

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Or via go install
go install github.com/mephistofox/fxtun.dev/cmd/fxtunnel@latest

# Verify
fxtunnel --version
```

### Frontend: React, Vue, Next.js on Port 3000

Most frontend frameworks run their dev server on port 3000. Open a tunnel to that port and your client can see your app in a browser.

```bash
# Start React / Vue / Next.js
npm run dev
# -> Local: http://localhost:3000

# In a second terminal — open a tunnel
fxtunnel http 3000
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://d7f2a.fxtun.dev
Forwarding:  https://d7f2a.fxtun.dev -> http://localhost:3000

Press Ctrl+C to stop
```

Send `https://d7f2a.fxtun.dev` to your client — they open the link and see your app. Hot reload keeps working: you make changes to the code, the client refreshes the page and sees the update.

#### Nuxt.js on Port 3000

```bash
npx nuxi dev
# -> Local: http://localhost:3000

fxtunnel http 3000
```

#### Vite on a Custom Port

```bash
npx vite --port 5173
# -> Local: http://localhost:5173

fxtunnel http 5173
```

### Backend API: Express on Port 8000

If you are demoing an API (for example, to a client who is a developer or to a frontend team), open a tunnel to the API server port.

```bash
# Express server
node server.js
# -> Listening on port 8000

fxtunnel http 8000
```

The client can test the API using a browser, Postman, or curl:

```bash
curl https://d7f2a.fxtun.dev/api/users
```

### Backend API: FastAPI on Port 8000

```bash
# FastAPI with uvicorn
uvicorn main:app --reload --port 8000
# -> Uvicorn running on http://127.0.0.1:8000

fxtunnel http 8000
```

FastAPI generates a Swagger UI automatically. The client can open `https://d7f2a.fxtun.dev/docs` and interactively test every endpoint.

### Full-Stack: Frontend + API Through One Tunnel

If your frontend and API run on different ports, there are two approaches.

#### Approach 1: Two Tunnels

```bash
# Terminal 1 — frontend
fxtunnel http 3000
# -> https://front-abc.fxtun.dev

# Terminal 2 — API
fxtunnel http 8000
# -> https://api-xyz.fxtun.dev
```

Update your frontend configuration so API requests go to the second tunnel's URL. In React, this is the `REACT_APP_API_URL` environment variable.

#### Approach 2: One Tunnel via Proxy

Configure your frontend dev server to proxy API requests. Then a single tunnel is enough.

**React (package.json):**
```json
{
  "proxy": "http://localhost:8000"
}
```

**Vue (vite.config.js):**
```javascript
export default defineConfig({
  server: {
    proxy: {
      '/api': 'http://localhost:8000'
    }
  }
})
```

**Next.js (next.config.js):**
```javascript
module.exports = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8000/api/:path*',
      },
    ]
  },
}
```

Now one tunnel to port 3000 covers both the frontend and the API:

```bash
fxtunnel http 3000
# -> https://demo-abc.fxtun.dev — frontend + API via proxy
```

## Tips for Professional Demos

### Custom Domain

A link like `https://abc123.fxtun.dev` works, but `https://demo.yourcompany.com` looks more professional. Custom domains are available in fxTunnel from $5/mo.

```bash
fxtunnel http 3000 --domain demo.yourcompany.com
```

The client sees a familiar domain instead of a random string of characters. This matters when working with larger clients.

### Stable URL

In fxTunnel SaaS, the URL stays the same even on the free tier. You can send the link to a client in the morning, restart the tunnel at lunch, and continue the demo in the evening — the address will not change.

### HTTPS Out of the Box

Every fxTunnel tunnel gets an HTTPS certificate automatically. The client does not have to dismiss browser warnings about an insecure connection. This is critical for demoing apps that use the geolocation API, camera, microphone, or Service Workers — browsers require HTTPS for these features. [HTTPS on localhost](/blog/https-localhost-development/) covers why this matters and how it works.

## For Freelancers: Impress Clients Without Staging Servers

If you are a freelancer, tunnels change the game. Here is a typical scenario:

1. The client messages you: "Can I see what you have so far?"
2. You run `fxtunnel http 3000` — 5 seconds.
3. You send the link in the chat.
4. The client opens it and sees the result. You discuss changes in real time.
5. You update the code — the client refreshes and sees the change.

No extra hosting costs. No time spent on deployment. The client gets instant access to the current version of the project.

With a custom domain (from $5/mo), the demo looks even more polished. Instead of `https://random.fxtun.dev`, the client sees `https://preview.yourname.dev` — this creates the impression of a well-oiled workflow.

### Demo Checklist for Freelancers

- Start your project locally and make sure everything works.
- Use test data, not the client's real data.
- Open a tunnel and verify the URL in a different browser.
- Send the link to the client with a brief description of what to look at.
- Close the tunnel (`Ctrl+C`) after the demo.

## For Teams: Quick Feedback Loops During Development

In team development, tunnels speed up the feedback cycle. Here are a few scenarios.

### Design Review

A designer wants to see how a mockup looks once implemented. Instead of deploying to staging, open a tunnel and drop the link in Slack.

```bash
fxtunnel http 3000
# Paste the URL into Slack: "Check out the mockup implementation: https://d7f2a.fxtun.dev"
```

### Microservice Integration

A backend developer updated the API. A frontend developer wants to test the integration before merging. The backend dev opens a tunnel to their API, and the frontend dev connects via the public URL.

```bash
# Backend developer
fxtunnel http 8000
# -> https://api-review.fxtun.dev

# Frontend developer updates API_URL in .env
# REACT_APP_API_URL=https://api-review.fxtun.dev
```

### QA Testing

A tester can check functionality on their own device (including mobile) without waiting for a staging deployment.

## Security: What to Watch Out For During Demos

A tunnel opens your local server to the internet, so you should treat it with the same care as any public-facing service. More on securing tunnels in [How to Expose Localhost](/blog/expose-localhost-to-internet/).

### Ground Rules

- **Close the tunnel after the demo.** Hit `Ctrl+C` when you are done. Do not leave it running overnight.
- **Use test data.** Do not demo the project with the client's real database. Populate the DB with seed records.
- **Do not hardcode secrets.** Make sure API keys and passwords are not in the code — use environment variables.
- **Do not expose admin panels.** If you are demoing the user-facing part, make sure `/admin` or `/dashboard` is behind authentication.

### What Not to Do

| Bad Practice | Why It Is Risky | What to Do Instead |
|---|---|---|
| Leaving the tunnel running overnight | Anyone with the URL gets access | Close with `Ctrl+C` after the demo |
| Demoing with a real database | Data leak risk | Use seed data |
| Hardcoded API keys | Exposed via inspector | `.env` + `.gitignore` |
| Open `/admin` route | Unauthorized access | Basic authentication |

### Extra Protection

If the demo lasts several days (for example, the client is testing on their own), take additional measures:

```bash
# Custom domain + HTTPS (from $5/mo)
fxtunnel http 3000 --domain preview.yoursite.com
```

It is also a good idea to add basic HTTP authentication at the application or reverse proxy level to restrict access to the client only.

## Comparison: Tunnel vs Other Demo Methods

| Method | Setup Time | Cost | HTTPS | Data Freshness |
|---|---|---|---|---|
| **fxTunnel** | 30 seconds | Free (custom domain from $5/mo) | Built-in | Always current (live) |
| Deploy to staging | 15-60 minutes | $5-50/mo for a server | Needs Let's Encrypt | After each deploy |
| Vercel / Netlify preview | 3-10 minutes | Free (with limits) | Yes | After push |
| Screenshots / video | 5-15 minutes | Free | N/A | At the time of recording |
| Screen-share call | 0 minutes | Free | N/A | Live, but no independent access |

A tunnel wins when the client needs independent access to the current version of the project without the delay of deployment. If you are also evaluating tunneling tools, [Free ngrok Alternatives](/blog/free-ngrok-alternatives-2026/) covers the main options.

## FAQ

### How do I show my local site to a client without deploying?

Run `fxtunnel http 3000` in your terminal. Within seconds you will have a public HTTPS URL (something like `https://abc123.fxtun.dev`) that you can paste into a chat or email. The client opens it in a browser and sees your running application — no servers, no DNS setup needed.

### Is it safe to share localhost through a tunnel?

Yes, with common-sense precautions. Traffic is encrypted via TLS. Stick to test data, shut the tunnel down with `Ctrl+C` when the demo is over, and avoid leaving it open overnight. If you need longer-running access, consider IP restrictions or basic HTTP authentication.

### Can I demo frontend and backend at the same time?

Sure. You can open two tunnels — `fxtunnel http 3000` for the frontend and `fxtunnel http 8000` for the API. Or, if you set up a dev-server proxy (like the `proxy` field in `package.json` for React), one tunnel on port 3000 covers both.

### Does the URL change when I restart the tunnel?

With fxTunnel SaaS, no — the URL persists across restarts, even on the free plan. If you use a custom domain (from $5/mo), you control the address entirely, for example `demo.yoursite.com`. Either way, the client keeps the same link.

### Can I use a tunnel to demo a mobile app?

Yes. If the mobile app talks to a local API, open a tunnel to that API port and swap `localhost` for the tunnel URL in your app config. The device can now reach your API from any network, not just the same Wi-Fi.
