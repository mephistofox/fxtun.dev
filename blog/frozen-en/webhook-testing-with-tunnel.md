---
title: "Webhook Testing on Localhost — Stripe, GitHub, Telegram"
date: 2026-02-03T14:00:00+03:00
draft: false
description: "Test webhooks on localhost with fxTunnel: step-by-step for Stripe, GitHub, Telegram. Inspector + replay for debugging and troubleshooting."
tags: [webhooks, testing, stripe, github, telegram, fxTunnel, localhost, tunnel, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Why do webhooks not reach localhost?"
    a: "Because localhost is an address that only exists inside your machine. External services like Stripe, GitHub, and Telegram cannot send an HTTP request to 127.0.0.1 — it is not routable over the internet. To receive webhooks, you need a public URL, which you can get through a local tunnel like fxTunnel."
  - q: "Is it safe to receive webhooks through a tunnel?"
    a: "For development and testing, yes. fxTunnel encrypts traffic via TLS. However, you should always verify the webhook signature on your application side to guard against forged requests. Do not use a tunnel for production webhooks."
  - q: "Do I need to change the webhook URL every time I restart the tunnel?"
    a: "With fxTunnel SaaS, the URL stays the same across restarts — even on the free tier. With a custom domain (from $5/mo), the URL is fully under your control. You do not need to update webhook settings after restarting the tunnel."
  - q: "How do I debug a webhook when no data is coming through?"
    a: "Use the built-in Inspector in fxTunnel (from $5/mo): it shows all incoming requests in real time with headers and body. Also check: 1) the tunnel is running and the URL is reachable in a browser, 2) the URL is correctly entered in the service settings, 3) your local server is listening on the right port."
  - q: "What is Inspector and Replay in fxTunnel?"
    a: "Inspector is a web interface for viewing all HTTP requests passing through the tunnel, including headers, body, and response status. Replay lets you resend any request with one click — perfect for debugging webhooks. Available on the plan from $5/mo."
---

## The Problem: Webhooks Need a Public URL

We have all been there -- you are building a Stripe integration, your endpoint is ready, and you realize Stripe cannot reach `localhost:8080`. A [local tunnel](/blog/what-is-tunneling/) fixes this in 30 seconds.

A webhook is an HTTP request that an external service sends to your server when an event occurs. Stripe notifies you about a payment, GitHub about a push, Telegram about a new message to your bot. The catch is that all of these services require a public URL. Your `localhost:8080` is not reachable from the internet, and pointing a webhook at `http://127.0.0.1` simply does not work.

The classic workarounds — deploying to a staging server after every change, or using request-capture services — are slow and clumsy. You spend minutes deploying just to verify a single line of code. And capture services show you the request body but do not let you step through your handling logic in a debugger.

## The Solution: A Local Tunnel for Webhook Testing

A local tunnel creates a public URL that points straight at your localhost. A webhook from Stripe hits `https://abc.fxtun.dev`, the [tunnel server](/blog/fxtunnel-architecture/) forwards the request to your `localhost:8080`, and your application receives the data in real time. The development cycle shrinks to seconds.

What this means for you as a developer:

- **Instant feedback** — you see webhooks the moment they are sent, with no deploy delay.
- **Inspector + Replay** — the built-in web interface shows all incoming requests with headers and body. Replay lets you resend any webhook with one click — no need to re-trigger the event in Stripe or GitHub. Available from $5/mo.
- **Real data** — test with actual events from Stripe, GitHub, or Telegram instead of hand-rolled mocks.
- **IDE debugging** — set breakpoints and inspect payloads right in your debugger. This is impossible when testing on a remote server.
- **No deployment** — everything runs on your local machine. Change a line of code and see the result immediately.
- **Security** — test data never leaves your machine. The tunnel encrypts traffic via TLS.

## Step-by-Step Guide: Testing Webhooks with fxTunnel

Setup takes 30 seconds: install the CLI, start a tunnel with one command, and paste the public URL into your webhook settings. No need to spin up a server, configure DNS, or manage certificates.

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify the installation
fxtunnel --version
```

### Step 2. Start Your Local Server

Make sure your application is running and listening on the correct port. For this example, we will use a simple Express server:

```javascript
// webhook-server.js
const express = require('express');
const app = express();

app.use(express.json());

app.post('/webhook', (req, res) => {
  console.log('Webhook received:', JSON.stringify(req.body, null, 2));
  console.log('Headers:', JSON.stringify(req.headers, null, 2));
  res.status(200).json({ received: true });
});

app.listen(8080, () => {
  console.log('Server listening on port 8080');
});
```

```bash
node webhook-server.js
# -> Server listening on port 8080
```

### Step 3. Create a Tunnel

```bash
# Open a tunnel to port 8080
fxtunnel http 8080
```

You will see the public URL:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://wh-demo.fxtun.dev
Forwarding:  https://wh-demo.fxtun.dev -> http://localhost:8080
```

Now `https://wh-demo.fxtun.dev/webhook` is your webhook endpoint.

### Step 4. Configure the Webhook in the Service

Enter the full URL `https://wh-demo.fxtun.dev/webhook` in the settings panel of the service you are integrating with. Details for specific services are below.

## Webhook Testing Examples for Popular Services

Here are ready-to-use examples for Stripe, GitHub, and Telegram. Each one covers endpoint configuration, signature verification, and event handling. All examples use the same tunnel public URL -- just run `fxtunnel http 8080`.

### Stripe: Testing Payment Webhooks

Stripe relies heavily on webhooks to notify you about payments, subscriptions, and disputes. Without webhooks, you have no way to learn the outcome of asynchronous operations — for example, a successful 3D Secure payment or a transaction dispute. When building a payment integration, webhook testing is a mandatory step.

#### Setup via Dashboard

1. Open the [Stripe Dashboard](https://dashboard.stripe.com/webhooks) in test mode.
2. Click **Add endpoint**.
3. Enter the URL: `https://wh-demo.fxtun.dev/webhook`.
4. Select events: `payment_intent.succeeded`, `checkout.session.completed`.
5. Save and copy the **Webhook Signing Secret** (`whsec_...`).

#### Signature Verification on the Server

```javascript
// Verify Stripe webhook signature
const stripe = require('stripe')('sk_test_...');
const endpointSecret = 'whsec_...';

app.post('/webhook', express.raw({ type: 'application/json' }), (req, res) => {
  const sig = req.headers['stripe-signature'];

  try {
    const event = stripe.webhooks.constructEvent(req.body, sig, endpointSecret);
    console.log('Stripe event:', event.type);

    // Handle specific events
    switch (event.type) {
      case 'payment_intent.succeeded':
        console.log('Payment succeeded:', event.data.object.id);
        break;
      case 'checkout.session.completed':
        console.log('Checkout completed:', event.data.object.id);
        break;
    }

    res.status(200).json({ received: true });
  } catch (err) {
    console.error('Signature verification failed:', err.message);
    res.status(400).send(`Webhook Error: ${err.message}`);
  }
});
```

#### Sending a Test Event

```bash
# Via Stripe CLI (an alternative way to test)
stripe trigger payment_intent.succeeded \
  --override payment_intent:metadata.order_id=test_123
```

### GitHub: Webhook for Push Events

#### Setup

1. Open your repository on GitHub, then go to **Settings** and click on **Webhooks**.
2. Click **Add webhook**.
3. Payload URL: `https://wh-demo.fxtun.dev/webhook`.
4. Content type: `application/json`.
5. Secret: create a secret for signature verification.
6. Select the event: **Just the push event**.

#### Server-Side Handler

```javascript
const crypto = require('crypto');

const GITHUB_SECRET = 'your_webhook_secret';

app.post('/webhook', (req, res) => {
  // Verify GitHub signature
  const signature = req.headers['x-hub-signature-256'];
  const hmac = crypto.createHmac('sha256', GITHUB_SECRET);
  const digest = 'sha256=' + hmac.update(JSON.stringify(req.body)).digest('hex');

  if (signature !== digest) {
    return res.status(401).send('Invalid signature');
  }

  const event = req.headers['x-github-event'];
  console.log(`GitHub event: ${event}`);

  if (event === 'push') {
    const { ref, commits, pusher } = req.body;
    console.log(`Push to ${ref} by ${pusher.name}`);
    console.log(`Commits: ${commits.length}`);
    commits.forEach(c => console.log(`  - ${c.message}`));
  }

  res.status(200).json({ ok: true });
});
```

### Telegram Bot: setWebhook

Telegram bots can operate in two modes: polling (the bot periodically asks the server for new messages) or webhooks (Telegram pushes messages to your URL). Webhooks are faster, more efficient, and use fewer resources. To develop a bot with webhooks, you need a public HTTPS address — exactly what a tunnel provides.

#### Registering the Webhook

```bash
# Set the webhook via the Telegram Bot API
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://wh-demo.fxtun.dev/webhook"}'

# Check the status
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

#### Handling Messages

```javascript
app.post('/webhook', (req, res) => {
  const { message } = req.body;

  if (message && message.text) {
    console.log(`Message from ${message.from.first_name}: ${message.text}`);

    // Reply to the user
    const chatId = message.chat.id;
    const replyText = `You said: ${message.text}`;

    fetch(`https://api.telegram.org/bot${BOT_TOKEN}/sendMessage`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ chat_id: chatId, text: replyText }),
    });
  }

  res.status(200).json({ ok: true });
});
```

#### Remove the Webhook After Testing

```bash
# Delete the webhook when you are done testing
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/deleteWebhook"
```

## Troubleshooting: Webhook Testing Issues with a Tunnel

Most webhook problems come down to three things: the tunnel is not running, the local server is not listening, or the URL is wrong. The Inspector in fxTunnel (from $5/mo) shows whether the request reached the tunnel, which immediately narrows down the issue.

| Problem | Possible cause | Solution |
|---|---|---|
| Webhook not arriving | Tunnel is not running | Make sure `fxtunnel` is running and the URL is reachable in a browser |
| HTTP 502 Bad Gateway | Local server is not running | Start your application on the correct port before creating the tunnel |
| HTTP 401 Unauthorized | Invalid webhook signature | Verify that the secret matches in both the service settings and your code |
| Request timeout | Handler takes too long | Respond with `200 OK` immediately and process the event asynchronously |
| Empty payload | Wrong Content-Type | Make sure `express.json()` middleware is loaded and the route path matches |
| URL changed after restart | Random subdomain | Use a custom domain in fxTunnel (from $5/mo) |
| Cannot see incoming data | No inspection tool | Open the fxTunnel Inspector — it shows all requests with headers and body |
| Need to resend a request | Event is hard to reproduce | Use Replay in fxTunnel — resend any request with one click |

## Webhook Testing Best Practices for Localhost Development

A solid approach to webhook testing saves hours of debugging. Here are five principles worth following from day one.

### 1. Always Verify the Signature

Do not skip signature verification even during development. Catching configuration errors early prevents surprises at deploy time.

### 2. Log Everything (or Use Inspector)

Instead of manual logging, you can use the Inspector in fxTunnel — it shows all incoming requests in a web interface with headers, body, and response status. But explicit server-side logging is still useful:

```javascript
// Log the full request for debugging
app.post('/webhook', (req, res) => {
  console.log('--- Incoming webhook ---');
  console.log('Method:', req.method);
  console.log('Path:', req.path);
  console.log('Headers:', JSON.stringify(req.headers, null, 2));
  console.log('Body:', JSON.stringify(req.body, null, 2));
  console.log('--- End ---');
  res.status(200).json({ received: true });
});
```

### 3. Respond Quickly

Most services expect a response within 5 to 30 seconds. If your processing takes longer, respond with `200 OK` right away and queue the work for background processing.

### 4. Handle Retries

Services will resend webhooks if they do not receive a `2xx` response. Implement idempotency: check the unique event ID before processing to avoid handling the same event twice.

### 5. Use Test Mode

Stripe, GitHub, and other services offer a test mode or the ability to send test events. Use these before configuring real data flows.

## Why fxTunnel Is the Best Tool for Webhook Testing

Here is how fxTunnel stacks up against other tools for webhook testing specifically. We go deeper in the [full comparison](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Feature | fxTunnel | Other tools |
|---|---|---|
| Free tier | No limits on traffic or connections | Often limited by time, requests, or connections |
| Inspector + Replay | From $5/mo — view all requests and resend with one click | Missing or requires a separate tool |
| Setup | One command, 30 seconds | Often requires configuration, tokens, registration |
| HTTP / TCP / UDP | All three protocols | Usually HTTP only |
| Custom domain | From $5/mo, up to 5 tunnels | More expensive or unavailable |
| 10+ tunnels | From $10/mo | Significantly more expensive |
| Open source | Yes, self-hosted option available | Often proprietary |

We also cover these tools in [Best Tunneling Tools 2026](/blog/best-tunneling-tools-2026/).

## FAQ

### Why do webhooks not reach localhost?

The address 127.0.0.1 (localhost) only exists inside your machine -- it is not routable over the public internet. Services like Stripe, GitHub, and Telegram have no way to send an HTTP request to it. You need a public URL that forwards to your local port, which is exactly what a [local tunnel](/blog/what-is-tunneling/) provides.

### Is it safe to receive webhooks through a tunnel?

For development and testing, absolutely. Traffic is encrypted via TLS, and you should always verify webhook signatures on your application side regardless of how the request arrives. Just keep in mind that tunnels are meant for dev workflows, not production webhook endpoints.

### Do I need to change the webhook URL every time I restart the tunnel?

With fxTunnel SaaS, the URL persists across restarts -- even on the free tier, so you do not need to update webhook settings every time. If you want a fully branded URL, custom domains are available from $5/mo.

### How do I debug a webhook when no data is coming through?

Start with the basics: is the tunnel running? Can you open the URL in a browser? Is the URL entered correctly in the service settings? Is your local server actually listening on the right port? The built-in Inspector in fxTunnel (from $5/mo) also helps by showing all incoming requests in real time with headers and body.

### What is Inspector and Replay in fxTunnel?

Inspector is a web UI that shows every HTTP request passing through the tunnel -- headers, body, response status, all of it. Replay lets you resend any captured request with one click, which is invaluable when you need to re-trigger a webhook without reproducing the original event. Both are available from $5/mo.
