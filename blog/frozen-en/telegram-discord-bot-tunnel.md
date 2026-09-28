---
title: "Telegram & Discord Bots via Tunnel — No Server"
date: 2026-03-19T14:00:00+03:00
draft: false
description: "Develop Telegram and Discord bots on localhost with fxTunnel: webhook setup, interaction endpoints, Inspector debugging, and Python/Node.js examples."
tags: [telegram, discord, bots, webhooks, fxTunnel, localhost, tunnel, python, nodejs, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Why does my Telegram bot need a public HTTPS URL?"
    a: "Telegram requires an HTTPS endpoint to deliver webhook updates. Your localhost is not reachable from the internet and does not have a TLS certificate. A tunnel like fxTunnel creates a public HTTPS URL that forwards requests to your local machine, so Telegram can push updates to your bot during development."
  - q: "Can I develop a Discord bot with webhooks on localhost?"
    a: "Yes. Discord Interactions Endpoint requires a publicly reachable HTTPS URL that responds to signature-verified POST requests. fxTunnel gives you that URL in seconds. Point your Discord application's Interactions Endpoint URL at the tunnel address and handle slash commands locally."
  - q: "Do I need to update the webhook URL every time I restart the tunnel?"
    a: "No. fxTunnel keeps the same URL across restarts — even on the free tier. With a custom domain (from $5/mo), the URL is fully under your control. You do not need to call setWebhook again or reconfigure Discord after restarting the tunnel."
  - q: "How do I debug webhook payloads from Telegram or Discord?"
    a: "Use the built-in Inspector in fxTunnel (from $5/mo). It shows every incoming request in real time with headers and body. The Replay feature lets you resend any request with one click — so you can re-test your handler without sending another message to the bot."
  - q: "Is it safe to receive bot webhooks through a tunnel?"
    a: "For development and testing, yes. fxTunnel encrypts traffic via TLS. Always verify the webhook signature on your application side — Telegram provides a secret token, and Discord uses Ed25519 signatures. Do not use a tunnel for production bots."
---

## The Problem: Bot Webhooks Need a Public URL

You are building a Telegram or Discord bot, and you want to use webhooks instead of polling. The catch? Your local server must be reachable from the internet. A [tunnel](/blog/what-is-tunneling/) solves this in seconds: you get a public HTTPS URL pointing at your localhost, so Telegram and Discord can push updates directly to your machine — no deployment, no VPS, no cloud functions.

Both Telegram and Discord offer a webhook-based model for receiving events. Telegram calls it `setWebhook` — instead of polling the API every few seconds, Telegram pushes each new message to your HTTPS endpoint. Discord uses an Interactions Endpoint — when a user invokes a slash command, Discord sends a POST request to your URL. In both cases, the platform needs a publicly accessible HTTPS address.

The problem is obvious: your `localhost:8080` is not reachable from the internet. You cannot point `setWebhook` at `http://127.0.0.1`. The classic workarounds — deploying to a cloud server after every code change, or using a long-polling fallback — are slow and limit your development workflow. You lose the ability to set breakpoints, inspect payloads in real time, and iterate quickly.

## The Solution: A Local Tunnel for Bot Development

One command — `fxtunnel http 8080` — and you have a public HTTPS URL pointing at your localhost. That URL works as an endpoint for Telegram `setWebhook` or Discord Interactions right away.

Here is what a tunnel-based workflow looks like:

1. Your bot server runs locally on `localhost:8080`.
2. fxTunnel creates a public URL like `https://bot-dev.fxtun.dev`.
3. You register `https://bot-dev.fxtun.dev/webhook` as the webhook URL in Telegram or Discord.
4. When a user sends a message or invokes a command, the platform sends a POST request to the tunnel URL.
5. The [tunnel server](/blog/fxtunnel-architecture/) forwards the request to your `localhost:8080`.
6. Your bot processes the event and responds.

What this means for you as a developer:

- **Instant feedback** — see bot messages and commands the moment they arrive, with no deploy delay.
- **IDE debugging** — set breakpoints and inspect payloads right in your debugger. This is impossible when running on a remote server.
- **Inspector + Replay** — the built-in web interface shows all incoming requests with headers and body. Replay lets you resend any webhook with one click — no need to type another message to the bot. We cover this in detail in the [Traffic Inspector](/blog/traffic-inspector-debug-requests/) article.
- **No deployment** — everything runs on your local machine. Change a line of code and see the result immediately.
- **Stable URL** — fxTunnel keeps the same URL across restarts, even on the free tier. No need to call `setWebhook` again.

## Quick Start: Install fxTunnel and Create a Tunnel

Install the CLI, start a tunnel, and you have a public URL for your bot.

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify the installation
fxtunnel --version
```

### Step 2. Start a Tunnel

```bash
# Open a tunnel to port 8080
fxtunnel http 8080
```

You will see the public URL:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://bot-dev.fxtun.dev
Forwarding:  https://bot-dev.fxtun.dev -> http://localhost:8080
```

Now `https://bot-dev.fxtun.dev` is your webhook endpoint. Keep this terminal open — the tunnel stays active as long as the process is running.

## Telegram Bot: Webhook Development with Python

Telegram bots can work in two modes: polling (the bot periodically asks the API for updates) and webhooks (Telegram pushes updates to your URL). Webhooks are faster, more efficient, and the recommended approach for production bots. During development, a tunnel lets you use webhooks from day one — so your code works identically in development and production.

### Setting the Webhook via curl

```bash
# Register the webhook URL with Telegram
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://bot-dev.fxtun.dev/webhook"}'

# Verify the webhook is set
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

The response from `getWebhookInfo` should show `"url": "https://bot-dev.fxtun.dev/webhook"` and `"has_custom_certificate": false` — fxTunnel handles TLS automatically.

### Python Bot with python-telegram-bot

Here is a complete Telegram bot using the `python-telegram-bot` library with webhook mode:

```python
# bot_telegram.py
import os
from telegram import Update
from telegram.ext import Application, CommandHandler, MessageHandler, filters, ContextTypes

BOT_TOKEN = os.environ["TELEGRAM_BOT_TOKEN"]
WEBHOOK_URL = "https://bot-dev.fxtun.dev/webhook"

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text(
        "Hello! I am running on localhost through an fxTunnel webhook."
    )

async def echo(update: Update, context: ContextTypes.DEFAULT_TYPE):
    text = update.message.text
    await update.message.reply_text(f"You said: {text}")

async def handle_error(update: Update, context: ContextTypes.DEFAULT_TYPE):
    print(f"Error: {context.error}")

def main():
    app = Application.builder().token(BOT_TOKEN).build()

    app.add_handler(CommandHandler("start", start))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))
    app.add_error_handler(handle_error)

    # Run in webhook mode on port 8080
    app.run_webhook(
        listen="0.0.0.0",
        port=8080,
        url_path="/webhook",
        webhook_url=WEBHOOK_URL,
    )

if __name__ == "__main__":
    main()
```

Run the bot:

```bash
# Install dependencies
pip install python-telegram-bot

# Set your bot token
export TELEGRAM_BOT_TOKEN="123456:ABC-DEF..."

# Start the bot
python bot_telegram.py
```

The `run_webhook` method does two things: it starts an HTTP server on port 8080 and calls `setWebhook` automatically. Combined with the fxTunnel tunnel, Telegram pushes updates directly to your machine.

### Node.js Telegram Bot with Express

If you prefer Node.js, here is the same bot using the `node-telegram-bot-api` package:

```javascript
// bot_telegram.js
const TelegramBot = require('node-telegram-bot-api');
const express = require('express');

const BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN;
const WEBHOOK_URL = 'https://bot-dev.fxtun.dev/webhook';

const bot = new TelegramBot(BOT_TOKEN);
const app = express();

app.use(express.json());

// Handle incoming updates from Telegram
app.post('/webhook', (req, res) => {
  bot.processUpdate(req.body);
  res.status(200).json({ ok: true });
});

// Command handlers
bot.onText(/\/start/, (msg) => {
  bot.sendMessage(msg.chat.id, 'Hello! I am running on localhost through an fxTunnel webhook.');
});

bot.on('message', (msg) => {
  if (msg.text && !msg.text.startsWith('/')) {
    bot.sendMessage(msg.chat.id, `You said: ${msg.text}`);
  }
});

// Start server and set webhook
app.listen(8080, async () => {
  console.log('Bot server listening on port 8080');
  await bot.setWebHook(WEBHOOK_URL);
  console.log(`Webhook set to ${WEBHOOK_URL}`);
});
```

```bash
npm install node-telegram-bot-api express

export TELEGRAM_BOT_TOKEN="123456:ABC-DEF..."
node bot_telegram.js
```

### Cleaning Up After Development

When you finish a development session, delete the webhook so Telegram does not keep sending requests to a URL that is no longer active:

```bash
# Remove the webhook
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/deleteWebhook"

# Switch back to polling if needed
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"
```

## Discord Bot: Interactions Endpoint on Localhost

Discord supports two models for receiving commands: a Gateway (persistent WebSocket connection) and an Interactions Endpoint (HTTP webhooks). The Interactions Endpoint is ideal for slash commands and message components — Discord sends a POST request to your URL, and your server responds synchronously. Like Telegram, Discord requires a public HTTPS endpoint with proper signature verification.

### How Discord Interactions Work

When a user invokes a slash command in Discord:

1. Discord sends a POST request to your Interactions Endpoint URL.
2. The request body contains the interaction data (command name, options, user info).
3. Your server verifies the Ed25519 signature using the public key from your Discord application.
4. Your server responds with the appropriate interaction response (message, deferred response, modal, etc.).

Discord also sends a `PING` interaction (type 1) when you save the Interactions Endpoint URL in the Developer Portal. Your server must respond with `{"type": 1}` to pass the verification check.

### Python Discord Bot with discord.py and Flask

```python
# bot_discord.py
import os
from flask import Flask, request, jsonify
from nacl.signing import VerifyKey
from nacl.exceptions import BadSignatureError

DISCORD_PUBLIC_KEY = os.environ["DISCORD_PUBLIC_KEY"]
DISCORD_BOT_TOKEN = os.environ["DISCORD_BOT_TOKEN"]

app = Flask(__name__)
verify_key = VerifyKey(bytes.fromhex(DISCORD_PUBLIC_KEY))

def verify_signature(req):
    """Verify Discord Ed25519 signature."""
    signature = req.headers.get("X-Signature-Ed25519")
    timestamp = req.headers.get("X-Signature-Timestamp")
    body = req.data.decode("utf-8")

    try:
        verify_key.verify(f"{timestamp}{body}".encode(), bytes.fromhex(signature))
        return True
    except (BadSignatureError, Exception):
        return False

@app.route("/discord", methods=["POST"])
def interactions():
    if not verify_signature(request):
        return "Invalid signature", 401

    data = request.json

    # Handle PING (type 1) — required for endpoint verification
    if data["type"] == 1:
        return jsonify({"type": 1})

    # Handle slash commands (type 2)
    if data["type"] == 2:
        command_name = data["data"]["name"]

        if command_name == "hello":
            return jsonify({
                "type": 4,  # CHANNEL_MESSAGE_WITH_SOURCE
                "data": {
                    "content": f"Hello, {data['member']['user']['username']}! "
                               f"This bot is running on localhost via fxTunnel."
                }
            })

        if command_name == "ping":
            return jsonify({
                "type": 4,
                "data": {"content": "Pong! Response from localhost."}
            })

    return jsonify({"type": 1})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
```

```bash
pip install flask pynacl

export DISCORD_PUBLIC_KEY="your_public_key_hex"
export DISCORD_BOT_TOKEN="your_bot_token"
python bot_discord.py
```

### Node.js Discord Bot with Express

```javascript
// bot_discord.js
const express = require('express');
const { verifyKey } = require('discord-interactions');

const DISCORD_PUBLIC_KEY = process.env.DISCORD_PUBLIC_KEY;

const app = express();

// Discord requires the raw body for signature verification
app.use(express.raw({ type: 'application/json' }));

app.post('/discord', (req, res) => {
  const signature = req.headers['x-signature-ed25519'];
  const timestamp = req.headers['x-signature-timestamp'];

  const isValid = verifyKey(req.body, signature, timestamp, DISCORD_PUBLIC_KEY);
  if (!isValid) {
    return res.status(401).send('Invalid signature');
  }

  const data = JSON.parse(req.body.toString());

  // Handle PING
  if (data.type === 1) {
    return res.json({ type: 1 });
  }

  // Handle slash commands
  if (data.type === 2) {
    const commandName = data.data.name;

    if (commandName === 'hello') {
      return res.json({
        type: 4,
        data: {
          content: `Hello, ${data.member.user.username}! This bot is running on localhost via fxTunnel.`,
        },
      });
    }

    if (commandName === 'ping') {
      return res.json({
        type: 4,
        data: { content: 'Pong! Response from localhost.' },
      });
    }
  }

  res.json({ type: 1 });
});

app.listen(8080, () => {
  console.log('Discord bot server listening on port 8080');
});
```

```bash
npm install express discord-interactions

export DISCORD_PUBLIC_KEY="your_public_key_hex"
node bot_discord.js
```

### Configuring the Interactions Endpoint in Discord Developer Portal

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications).
2. Select your application.
3. Navigate to **General Information**.
4. In the **Interactions Endpoint URL** field, enter: `https://bot-dev.fxtun.dev/discord`.
5. Click **Save Changes**.

Discord will send a PING request to verify your endpoint. If your server is running behind the fxTunnel tunnel and responds correctly, the URL is saved. If verification fails, check that your server is running, the tunnel is active, and the signature verification code is correct.

### Registering Slash Commands

Before users can invoke your commands, you need to register them with Discord:

```bash
# Register a global slash command
curl -X POST "https://discord.com/api/v10/applications/<APP_ID>/commands" \
  -H "Authorization: Bot <BOT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello",
    "description": "Say hello from the bot",
    "type": 1
  }'

curl -X POST "https://discord.com/api/v10/applications/<APP_ID>/commands" \
  -H "Authorization: Bot <BOT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ping",
    "description": "Check bot responsiveness",
    "type": 1
  }'
```

Global commands take up to an hour to propagate. For faster testing, register guild-specific commands by adding `/guilds/<GUILD_ID>` to the URL — they appear instantly.

## Debugging Bot Webhooks with Inspector

Ever had your bot silently ignore a command with no clue why? The Inspector in fxTunnel shows every incoming request in real time — headers, body, status code, and timing. When your bot does not respond as expected, you see exactly what the platform sent and what your server returned.

### Typical Debugging Scenario

1. A user sends `/start` to your Telegram bot.
2. The bot does not reply.
3. Open Inspector at `https://bot-dev.fxtun.dev/_inspector`.
4. You see the incoming POST request with the full Telegram Update payload.
5. The response shows `500 Internal Server Error` with the error message.
6. You fix the bug, click **Replay** to resend the exact same request.
7. The response now shows `200 OK`. Problem solved.

Without Inspector, you would need to send another message to the bot, wait for Telegram to deliver it, and hope to reproduce the same conditions. With Replay, you re-test the exact same payload as many times as needed.

### What Inspector Shows for Bot Webhooks

For a Telegram update:

```
POST /webhook HTTP/1.1
Content-Type: application/json

{
  "update_id": 123456789,
  "message": {
    "message_id": 42,
    "from": {"id": 100, "first_name": "Alex"},
    "chat": {"id": 100, "type": "private"},
    "text": "/start"
  }
}

--- Response ---
HTTP/1.1 200 OK
{"ok": true}
```

For a Discord interaction:

```
POST /discord HTTP/1.1
Content-Type: application/json
X-Signature-Ed25519: abc123...
X-Signature-Timestamp: 1711022400

{
  "type": 2,
  "data": {"name": "hello"},
  "member": {"user": {"username": "Alex"}}
}

--- Response ---
HTTP/1.1 200 OK
{"type": 4, "data": {"content": "Hello, Alex! ..."}}
```

You see exactly what your server receives and what it returns — no guessing, no manual logging. The [Traffic Inspector](/blog/traffic-inspector-debug-requests/) article covers all the capabilities in depth.

## Comparison: Polling vs Webhooks for Bot Development

Both Telegram and Discord offer an alternative to webhooks. Understanding the trade-offs helps you choose the right approach for your project.

| Aspect | Polling / Gateway | Webhooks + Tunnel |
|---|---|---|
| **Setup complexity** | Minimal — no public URL needed | Requires a tunnel or public server |
| **Latency** | Higher — periodic API calls or reconnection delays | Lower — instant delivery from the platform |
| **Resource usage** | Higher — constant connection or repeated requests | Lower — server only processes actual events |
| **Debugging** | Limited — need to add logging manually | Inspector shows every request in real time |
| **Production readiness** | Polling is not recommended for production Telegram bots | Webhook code works identically in dev and prod |
| **IDE breakpoints** | Works locally | Works locally with a tunnel |
| **Replay** | Must re-trigger the event manually | One-click Replay in Inspector |

For development, webhooks with a tunnel give you the closest match to production behavior plus better debugging tools. The code you write during development deploys to production without changes — just swap the tunnel URL for your production domain.

## Running Both Bots Simultaneously

If you are developing both a Telegram and a Discord bot (or any combination of services), you can run multiple tunnels at once. fxTunnel supports multiple simultaneous tunnels on the free tier.

### Option 1: Different Ports, Separate Tunnels

```bash
# Terminal 1: Telegram bot on port 8080
python bot_telegram.py  # listens on 8080
fxtunnel http 8080

# Terminal 2: Discord bot on port 8090
python bot_discord.py   # listens on 8090
fxtunnel http 8090
```

### Option 2: Single Server, Multiple Routes

Run both bots on the same server with different URL paths:

```python
# bot_combined.py
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route("/telegram", methods=["POST"])
def telegram_webhook():
    update = request.json
    # Handle Telegram update...
    return jsonify({"ok": True})

@app.route("/discord", methods=["POST"])
def discord_webhook():
    data = request.json
    # Handle Discord interaction...
    return jsonify({"type": 1})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
```

```bash
# One tunnel, two endpoints
fxtunnel http 8080

# Telegram webhook -> https://bot-dev.fxtun.dev/telegram
# Discord endpoint -> https://bot-dev.fxtun.dev/discord
```

This approach uses a single tunnel and a single port — simpler to manage and monitor.

## Troubleshooting: Common Bot Webhook Issues

Something not working? Most bot webhook problems come down to three things: the tunnel is not running, the signature verification fails, or the response format is wrong. The [Inspector](/blog/traffic-inspector-debug-requests/) can tell you whether the request even reached your server — that alone narrows down the issue instantly.

| Problem | Possible cause | Solution |
|---|---|---|
| Telegram: `setWebhook` returns error | URL not reachable | Make sure the tunnel is running and the URL opens in a browser |
| Telegram: bot does not respond | Wrong port or path | Verify that `url_path` in your code matches the webhook URL path |
| Telegram: `getWebhookInfo` shows errors | Server returns non-200 | Check Inspector for the response status code and error message |
| Discord: endpoint verification fails | Signature check broken | Ensure you use the raw body (not parsed JSON) for Ed25519 verification |
| Discord: PING not acknowledged | Missing type 1 handler | Your server must respond with `{"type": 1}` to PING interactions |
| Discord: slash command not appearing | Command not registered | Register commands via the Discord API; global commands take up to 1 hour |
| HTTP 502 Bad Gateway | Bot server not running | Start your application before creating the tunnel |
| URL changed after restart | Random subdomain | fxTunnel keeps the same URL across restarts; use a custom domain for full control |

## Best Practices for Bot Development with a Tunnel

A structured approach to bot development saves hours of debugging. These practices apply to both Telegram and Discord bots.

### 1. Verify Signatures from Day One

Telegram provides a `secret_token` parameter in `setWebhook` to verify incoming requests. Discord requires Ed25519 signature verification. Implement both from the start — skipping verification during development means you will forget it before production.

### 2. Use Inspector Instead of Print Debugging

Instead of scattering `print(request.json)` through your code, open the Inspector. It shows every request and response with proper formatting, and Replay lets you re-test without triggering another event.

### 3. Respond Quickly

Telegram expects a response within 60 seconds. Discord is stricter — your server must respond within 3 seconds. For long-running tasks, use Discord's deferred response pattern (type 5) and follow up with a webhook edit.

### 4. Handle Errors Gracefully

If your bot crashes, the platform will retry the webhook. Implement proper error handling so your server always returns a valid response — even when the processing logic fails.

### 5. Separate Bot Logic from HTTP Handling

Keep your webhook route handlers thin. Parse the incoming data, dispatch to the appropriate handler function, and return the response. This makes your code testable and portable between webhook and polling modes.

### 6. Use Environment Variables

Never hardcode bot tokens, public keys, or webhook URLs. Use environment variables so you can switch between development (tunnel URL) and production (your domain) without changing code.

## Why fxTunnel for Bot Development

What matters for bot development specifically? A stable URL (so you do not re-register webhooks), a traffic inspector (so you can see what the platform sent), and a free tier that does not get in the way. Here is how fxTunnel stacks up against [other tunneling tools](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Feature | fxTunnel | Other tools |
|---|---|---|
| Free tier | No limits on traffic or connections | Often limited by time, requests, or connections |
| Stable URL | Same URL across restarts, even on free tier | Often changes on restart |
| Inspector + Replay | From $5/mo — view all requests and resend with one click | Missing or requires a separate tool |
| Custom domain | From $5/mo, up to 5 tunnels | More expensive or unavailable |
| 10+ tunnels | From $10/mo | Significantly more expensive |
| HTTPS | Automatic TLS — required by Telegram and Discord | Usually included |
| Setup | One command, 30 seconds | Often requires tokens, registration, configuration |

You can also check out the [best tunneling tools 2026](/blog/best-tunneling-tools-2026/) overview and the guide on [exposing localhost to the internet](/blog/expose-localhost-to-internet/) with security best practices.

## FAQ

### Why does my Telegram bot need a public HTTPS URL?

Telegram delivers webhook updates over HTTPS and will not send them to a private address. Since your localhost has no public IP and no TLS certificate, you need a [tunnel](/blog/what-is-tunneling/) to bridge the gap. fxTunnel gives your local machine a public HTTPS URL so Telegram can reach your bot while you develop.

### Can I develop a Discord bot with webhooks on localhost?

Absolutely. Discord's Interactions Endpoint expects a public HTTPS URL that can handle signature-verified POST requests. A tunnel provides exactly that. Set your Discord application's Interactions Endpoint URL to the tunnel address, and you can process slash commands right on your machine.

### Do I need to update the webhook URL every time I restart the tunnel?

No -- fxTunnel preserves the same URL between restarts, even on the free tier. With a custom domain the URL never changes at all. There is no need to call `setWebhook` again or touch Discord settings after a restart.

### How do I debug webhook payloads from Telegram or Discord?

The Inspector in fxTunnel captures every incoming request with full headers and body as it arrives. If something goes wrong, you can hit Replay to resend the exact same payload without messaging the bot again. The [Traffic Inspector](/blog/traffic-inspector-debug-requests/) article walks through the workflow in detail.

### Is it safe to receive bot webhooks through a tunnel?

For development and testing, yes -- fxTunnel encrypts all traffic with TLS. That said, always verify webhook signatures in your code: Telegram supports a secret token, and Discord uses Ed25519. Tunnels are meant for development; do not use them for production bots. See also our [webhook testing best practices](/blog/webhook-testing-with-tunnel/).
