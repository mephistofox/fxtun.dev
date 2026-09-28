---
title: "WordPress on Localhost via Tunnel — Test & Demo"
date: 2026-03-15T14:00:00+03:00
draft: false
description: "Develop and test WordPress on localhost with fxTunnel: WP_HOME/WP_SITEURL config, plugin testing, client demos, and WooCommerce webhooks."
tags: [wordpress, localhost, tunnel, fxTunnel, woocommerce, webhooks, development, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Why does WordPress break when I open it through a tunnel?"
    a: "WordPress hardcodes its URL in the database (siteurl and home options). If you open the site through a tunnel URL that does not match, WordPress redirects you back to the original address, stylesheets fail to load, and links break. The fix is to override WP_HOME and WP_SITEURL in wp-config.php to match the tunnel URL — or use a filter that sets them dynamically from the request host."
  - q: "Can I use a tunnel to demo a WordPress site to a client?"
    a: "Yes. Start your local WordPress, open a tunnel with fxtunnel http 8080, update WP_HOME and WP_SITEURL to the tunnel URL, and send the link to your client. They will see exactly what is on your machine — theme, plugins, content — without any deployment. Close the tunnel with Ctrl+C when the demo is over."
  - q: "How do I test WooCommerce webhooks on localhost?"
    a: "Create a tunnel with fxtunnel http 8080 and use the tunnel URL as the webhook Delivery URL in WooCommerce settings. WooCommerce will send webhook notifications (order created, payment completed, etc.) to the tunnel, which forwards them to your local WordPress. Use the fxTunnel Inspector (from $5/mo) to view request payloads."
  - q: "Does the tunnel URL change when I restart it?"
    a: "With fxTunnel SaaS, the URL stays the same across restarts — even on the free tier. With a custom domain (from $5/mo), you get a fully permanent address like dev.yoursite.com. You do not need to update wp-config.php or webhook settings after restarting the tunnel."
  - q: "Is it safe to expose my WordPress development site through a tunnel?"
    a: "For development and short demos, yes. fxTunnel encrypts all traffic via TLS, so the connection is secure. Use test data, keep wp-admin behind strong credentials, and close the tunnel when you are done. Do not expose a production WordPress site through a tunnel."
---

## The Problem: WordPress Is Trapped on Localhost

WordPress development on localhost is convenient — until you need to show your work to someone else. Your client cannot open `http://localhost:8080`. Payment gateways cannot send webhooks to `127.0.0.1`. A plugin that relies on external OAuth callbacks has no public URL to redirect to. And deploying to a staging server after every CSS tweak is a waste of time.

On top of that, WordPress has a unique problem: it hardcodes its URL in the database. The `siteurl` and `home` options control how WordPress generates links, loads assets, and handles redirects. If you simply point a [tunnel](/blog/what-is-tunneling/) at your local WordPress without updating these values, the site will break — stylesheets will not load, links will redirect to `localhost`, and the admin panel will become inaccessible.

This article shows you how to set up a proper localhost-to-tunnel workflow for WordPress: configure `WP_HOME` and `WP_SITEURL`, test plugins and themes with live traffic, demo sites to clients, and debug WooCommerce webhooks — all without deploying.

## The Solution: A Tunnel with Proper WordPress Configuration

fxTunnel creates a public HTTPS URL for your local WordPress in one command. Combined with a two-line `wp-config.php` change, you get a fully functional WordPress site accessible from anywhere on the internet.

Here is the workflow:

1. Run WordPress locally (XAMPP, Docker, wp-env, Local — anything).
2. Start a tunnel with `fxtunnel http 8080`.
3. Set `WP_HOME` and `WP_SITEURL` to the tunnel URL.
4. Open the tunnel URL in a browser — your WordPress site works with full HTTPS.

If you are new to tunneling, the [What Is Tunneling](/blog/what-is-tunneling/) article covers the fundamentals.

## Step-by-Step Guide: WordPress + fxTunnel

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Or via go install
go install github.com/mephistofox/fxtun.dev/cmd/fxtunnel@latest

# Verify the installation
fxtunnel --version
```

### Step 2. Start Your Local WordPress

Use whatever local environment you prefer. Here are the most common options:

**Docker Compose (recommended):**
```yaml
# docker-compose.yml
version: '3.8'
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: wordpress
      WORDPRESS_DB_NAME: wordpress
    volumes:
      - ./wp-content:/var/www/html/wp-content

  db:
    image: mysql:8.0
    environment:
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: wordpress
      MYSQL_ROOT_PASSWORD: rootpassword
    volumes:
      - db_data:/var/lib/mysql

volumes:
  db_data:
```

```bash
docker compose up -d
# -> WordPress is running on http://localhost:8080
```

**wp-env (for plugin/theme development):**
```bash
# Install wp-env globally
npm install -g @wordpress/env

# Start WordPress in the current plugin/theme directory
wp-env start
# -> WordPress is running on http://localhost:8888
```

**XAMPP / MAMP / Local by Flywheel:** start your environment as usual and note the port.

### Step 3. Create a Tunnel

```bash
# Open a tunnel to your WordPress port
fxtunnel http 8080
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://wp-demo.fxtun.dev
Forwarding:  https://wp-demo.fxtun.dev -> http://localhost:8080
```

Now `https://wp-demo.fxtun.dev` points to your local WordPress. But if you open this URL right now, WordPress will likely redirect you to `localhost:8080` or load without styles. That is because of the `siteurl` and `home` problem — let us fix it.

### Step 4. Configure WP_HOME and WP_SITEURL

Open `wp-config.php` and add two lines **before** the line that says `/* That's all, stop editing! */`:

```php
// wp-config.php — add these lines for tunnel access
define('WP_HOME', 'https://wp-demo.fxtun.dev');
define('WP_SITEURL', 'https://wp-demo.fxtun.dev');
```

These constants override the values stored in the database. WordPress will now generate all links, load all assets, and handle all redirects using the tunnel URL.

#### Dynamic Configuration (Better for Development)

If you do not want to edit `wp-config.php` every time you restart the tunnel (or if you switch between local and tunnel access frequently), use a dynamic approach:

```php
// wp-config.php — dynamic URL based on the request
if (isset($_SERVER['HTTP_HOST'])) {
    $scheme = (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https')
        ? 'https'
        : (isset($_SERVER['HTTPS']) && $_SERVER['HTTPS'] === 'on' ? 'https' : 'http');
    $host = $_SERVER['HTTP_HOST'];
    define('WP_HOME', "{$scheme}://{$host}");
    define('WP_SITEURL', "{$scheme}://{$host}");
}
```

This snippet reads the hostname and protocol from the incoming request. When you access the site through the tunnel, WordPress uses the tunnel URL. When you access it directly on `localhost:8080`, it uses `localhost:8080`. No manual changes needed.

#### Why You Need X-Forwarded-Proto

fxTunnel terminates TLS at the tunnel server and forwards requests to your local WordPress over plain HTTP. WordPress sees `HTTP`, not `HTTPS`. Without the `HTTP_X_FORWARDED_PROTO` check, WordPress may generate `http://` links even though the client is connected via `https://`. The dynamic snippet above handles this automatically.

You should also add this to `wp-config.php` if you are behind a reverse proxy (which a tunnel effectively is):

```php
// Trust the X-Forwarded-Proto header from the tunnel
if (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https') {
    $_SERVER['HTTPS'] = 'on';
}
```

### Step 5. Verify Everything Works

Open `https://wp-demo.fxtun.dev` in your browser. You should see:

- The green padlock icon (valid HTTPS certificate).
- Your WordPress theme with all styles loaded.
- Working navigation — all links point to `https://wp-demo.fxtun.dev/...`.
- The admin panel at `https://wp-demo.fxtun.dev/wp-admin/` loads correctly.

If something is off, see the Troubleshooting section below.

## Use Case 1: Client Demos Without Deploying

Showing a WordPress site to a client usually means deploying to a staging server — uploading files, migrating the database, fixing URLs, configuring SSL. With a tunnel, you skip all of that.

The workflow is simple:

1. Get the site looking right on your local machine.
2. Run `fxtunnel http 8080`.
3. Set `WP_HOME` and `WP_SITEURL` to the tunnel URL (or use the dynamic snippet).
4. Send the link to your client.

The client sees exactly what is on your screen — the same theme, the same content, the same plugins. You make changes, the client refreshes the page. No deploy cycle, no staging server costs.

We cover more demo scenarios in [How to Demo Your Local Project Without Deploying](/blog/demo-localhost-without-deploy/).

### Custom Domain for Professional Demos

A link like `https://wp-demo.fxtun.dev` works, but `https://preview.clientsite.com` looks more professional. Custom domains are available from $5/mo.

```bash
fxtunnel http 8080 --domain preview.clientsite.com
```

Update `wp-config.php` accordingly:

```php
define('WP_HOME', 'https://preview.clientsite.com');
define('WP_SITEURL', 'https://preview.clientsite.com');
```

## Use Case 2: Plugin and Theme Testing with Real Traffic

Have you ever tried testing an OAuth login plugin on localhost? It does not work -- there is no public URL for the redirect. The same goes for payment gateways, social sharing buttons, embeds, and CDN integrations. A tunnel gives you that public URL without deployment.

### Testing OAuth Plugins

Plugins like **Nextend Social Login**, **WP Social Login**, or **Login with Google** require an OAuth redirect URI. On localhost, you cannot register a valid redirect. With a tunnel:

```bash
fxtunnel http 8080
# -> https://wp-demo.fxtun.dev
```

Register `https://wp-demo.fxtun.dev/wp-admin/admin-ajax.php` (or whatever the plugin uses) as the redirect URI in your OAuth provider (Google, Facebook, GitHub). The login flow works end-to-end on your local machine. The [OAuth Callback on Localhost](/blog/oauth-callback-localhost-tunnel/) article covers this pattern in depth.

### Testing SEO and Social Sharing

When you share a WordPress page on social media, platforms like Facebook, Twitter, and LinkedIn fetch Open Graph meta tags from the URL. On localhost, they cannot reach your site. With a tunnel, the public URL is fetchable — you can test how your pages look when shared.

```bash
# Test Open Graph tags
curl -A "facebookexternalhit/1.1" https://wp-demo.fxtun.dev/sample-page/
```

### Testing Cron and Scheduled Tasks

WordPress uses `wp-cron.php` which is triggered by page visits. Some developers use external cron services that call `wp-cron.php` via HTTP. With a tunnel, those services can reach your local WordPress:

```bash
# External cron service can now hit this URL
# https://wp-demo.fxtun.dev/wp-cron.php?doing_wp_cron
```

## Use Case 3: WooCommerce Webhook Testing

WooCommerce relies on webhooks to notify external systems about orders, payments, inventory changes, and customer events. When building a custom integration — say, syncing orders with an ERP or sending notifications to a Slack channel — you need to test webhooks with real data. On localhost, WooCommerce has no public URL to deliver webhooks to.

### Setting Up WooCommerce Webhooks with a Tunnel

```bash
# Start the tunnel
fxtunnel http 8080
# -> https://wp-demo.fxtun.dev
```

1. Open **WooCommerce -> Settings -> Advanced -> Webhooks** in your WordPress admin.
2. Click **Add webhook**.
3. Name: `Test order webhook`.
4. Status: **Active**.
5. Topic: **Order created**.
6. Delivery URL: `https://wp-demo.fxtun.dev/wp-json/custom/v1/order-hook` (or your external endpoint through the tunnel).
7. Secret: generate a secret for signature verification.
8. Save.

Now create a test order in WooCommerce. The webhook fires and hits your Delivery URL — which the tunnel forwards to your local handler.

### Webhook Handler Example

If your webhook endpoint is on the same local WordPress (for example, a custom plugin endpoint), you can test the round trip:

```php
// In your custom plugin — register a REST API endpoint for the webhook
add_action('rest_api_init', function () {
    register_rest_route('custom/v1', '/order-hook', [
        'methods'  => 'POST',
        'callback' => 'handle_woo_webhook',
        'permission_callback' => '__return_true',
    ]);
});

function handle_woo_webhook(WP_REST_Request $request) {
    $payload = $request->get_json_params();
    $signature = $request->get_header('X-WC-Webhook-Signature');

    // Verify the signature
    $secret = 'your_webhook_secret';
    $expected = base64_encode(hash_hmac('sha256', $request->get_body(), $secret, true));

    if ($signature !== $expected) {
        return new WP_REST_Response(['error' => 'Invalid signature'], 401);
    }

    // Log the webhook data
    error_log('WooCommerce webhook received: ' . wp_json_encode($payload));

    // Process the order
    $order_id = $payload['id'] ?? null;
    $status = $payload['status'] ?? 'unknown';
    error_log("Order #{$order_id} — status: {$status}");

    return new WP_REST_Response(['received' => true], 200);
}
```

### External Webhook Receiver

If your webhook receiver is a separate application (Node.js, Python, etc.) running on a different port, open a second tunnel:

```bash
# Terminal 1 — WordPress
fxtunnel http 8080
# -> https://wp-demo.fxtun.dev

# Terminal 2 — webhook receiver (Node.js on port 3000)
fxtunnel http 3000
# -> https://hook-receiver.fxtun.dev
```

Use `https://hook-receiver.fxtun.dev/webhook` as the Delivery URL in WooCommerce. Or, if both services run locally, you can point the webhook at `http://localhost:3000/webhook` and skip the second tunnel — but the first tunnel is still needed for the WooCommerce admin UI.

### Debugging Webhooks with Inspector

The fxTunnel Inspector (from $5/mo) shows every HTTP request passing through the tunnel — headers, body, status code. This is invaluable for webhook debugging: you can see exactly what WooCommerce sent and how your handler responded. Replay lets you resend any request with one click, so you do not need to create a new test order every time.

The [Webhook Testing with a Tunnel](/blog/webhook-testing-with-tunnel/) article walks through this in more detail.

## Use Case 4: Multisite and Multi-Developer Workflows

### WordPress Multisite

WordPress Multisite uses domain-based or subdirectory-based URLs for each subsite. With a tunnel, you can test the main site, but subsites require additional configuration. The simplest approach is subdirectory-based Multisite:

```php
// wp-config.php for Multisite through a tunnel
define('WP_HOME', 'https://wp-demo.fxtun.dev');
define('WP_SITEURL', 'https://wp-demo.fxtun.dev');
define('DOMAIN_CURRENT_SITE', 'wp-demo.fxtun.dev');
```

Subdirectory subsites (`https://wp-demo.fxtun.dev/site2/`) will work through the tunnel without extra configuration.

### Sharing with Other Developers

When two developers need to test an integration (for example, a frontend developer needs the WordPress REST API), the backend developer opens a tunnel and shares the URL:

```bash
# Backend developer
fxtunnel http 8080
# -> https://wp-api.fxtun.dev

# Frontend developer uses the API
curl https://wp-api.fxtun.dev/wp-json/wp/v2/posts
```

The frontend developer can point their React or Vue app at `https://wp-api.fxtun.dev` as the API base URL and test the full integration without deploying.

## Local Development Environments: Comparison with Tunnel Support

Different local WordPress environments have different default ports and configurations. Here is how they work with fxTunnel:

| Environment | Default Port | Tunnel Command | Notes |
|---|---|---|---|
| **Docker Compose** | 8080 (configurable) | `fxtunnel http 8080` | Most flexible, full control over PHP/MySQL versions |
| **wp-env** | 8888 | `fxtunnel http 8888` | Official WordPress tool, great for plugin development |
| **XAMPP** | 80 | `fxtunnel http 80` | Classic stack, works on Windows/macOS/Linux |
| **MAMP** | 8888 | `fxtunnel http 8888` | Popular on macOS, has a GUI |
| **Local (Flywheel)** | varies | `fxtunnel http <port>` | GUI-based, check the site details for the port |
| **Lando** | varies | `fxtunnel http <port>` | Docker-based, check `lando info` for the port |
| **Laravel Valet** | 80 (via .test domain) | `fxtunnel http 80` | Lightweight, macOS only |

## Troubleshooting: Common WordPress + Tunnel Issues

Most problems come down to three things: `WP_HOME`/`WP_SITEURL` mismatch, HTTPS/HTTP confusion, or the local server not running. Use the table below to diagnose issues quickly.

| Problem | Cause | Solution |
|---|---|---|
| Redirect loop to localhost | `WP_HOME`/`WP_SITEURL` still set to `localhost` | Update `wp-config.php` with the tunnel URL |
| Styles and scripts not loading | Mixed content (HTTP assets on HTTPS page) | Add the `$_SERVER['HTTPS'] = 'on'` fix for X-Forwarded-Proto |
| Admin panel inaccessible | `WP_SITEURL` does not match the tunnel URL | Set `WP_SITEURL` to the exact tunnel URL, including `https://` |
| "Too many redirects" error | WordPress forces HTTPS but tunnel re-forwards | Add `$_SERVER['HTTPS'] = 'on'` when `X-Forwarded-Proto` is `https` |
| Images show broken URLs | `wp_get_attachment_url` returns localhost URL | `WP_HOME` must match the tunnel URL |
| REST API returns 401 | Nonce mismatch due to domain change | Log out and log back in through the tunnel URL |
| WooCommerce webhook fails | Delivery URL is still set to localhost | Update the webhook Delivery URL to the tunnel URL |
| Slow page load | Large site with many assets | Normal — assets travel through the tunnel. Use browser caching |

### Force HTTPS Fix

If WordPress plugins or themes check `is_ssl()` and it returns `false` because the tunnel forwards HTTP, add this to the top of `wp-config.php`:

```php
// Fix SSL detection behind a tunnel/reverse proxy
if (
    (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https') ||
    (isset($_SERVER['HTTP_X_FORWARDED_SSL']) && $_SERVER['HTTP_X_FORWARDED_SSL'] === 'on')
) {
    $_SERVER['HTTPS'] = 'on';
}
```

## WordPress Tunnel vs Staging Server: When to Use What

A tunnel is not a replacement for a staging server — it is a complement. Use it during active development and for quick demos. Use staging for long-running QA, client UAT, and pre-production testing.

| Criterion | fxTunnel | Staging Server |
|---|---|---|
| **Setup time** | 30 seconds | 30 minutes to several hours |
| **Cost** | Free (custom domain from $5/mo) | $5-50/mo for hosting |
| **HTTPS** | Automatic, valid certificate | Needs Let's Encrypt or manual setup |
| **Data freshness** | Always current (live localhost) | Snapshot from last deploy |
| **Collaboration** | One developer at a time (their machine) | Shared, team-accessible |
| **Persistence** | As long as the tunnel is running | Always on |
| **Performance** | Depends on your machine + network | Depends on server specs |
| **Best for** | Development, quick demos, webhook testing | QA, UAT, pre-production |

A broader comparison of demo methods is available in [How to Demo Without Deploying](/blog/demo-localhost-without-deploy/).

## Best Practices for WordPress + Tunnel Development

### 1. Use the Dynamic WP_HOME Snippet

Do not hardcode the tunnel URL — use the dynamic snippet from Step 4. This way you can switch between `localhost` and the tunnel URL without editing config files.

### 2. Keep wp-admin Secure

When your WordPress site is publicly accessible through a tunnel, `wp-admin` is also accessible. Use strong passwords and consider adding HTTP basic authentication on top:

```php
// .htaccess — basic auth for wp-admin (Apache)
<Files wp-login.php>
  AuthType Basic
  AuthName "Restricted"
  AuthUserFile /path/to/.htpasswd
  Require valid-user
</Files>
```

### 3. Use Test Data

Never demo a WordPress site with real customer data. Use [FakerPress](https://wordpress.org/plugins/fakerpress/) or WP CLI to generate test content:

```bash
# Generate test posts with WP CLI
wp post generate --count=20 --post_type=post --post_status=publish
wp user generate --count=5 --role=subscriber
```

### 4. Close the Tunnel After Use

Hit `Ctrl+C` when the demo or testing session is over. Do not leave the tunnel running overnight with a WordPress site behind it — `wp-admin` would be accessible to anyone with the URL.

### 5. Use Inspector for Debugging

The fxTunnel Inspector (from $5/mo) is especially useful with WordPress because WordPress sends many internal HTTP requests (REST API calls, cron, update checks). Inspector lets you see all of them and quickly spot misconfigured URLs or failed requests.

## Why fxTunnel for WordPress Development

WordPress sites tend to load many assets per page, so a tunnel without bandwidth caps matters. fxTunnel's built-in [HTTPS](/blog/https-localhost-development/) with a valid certificate means WordPress HTTPS features (Secure Cookies, `is_ssl()`, force-SSL admin) work correctly through the tunnel. And the URL stays the same across restarts, so you do not need to update `wp-config.php` every time.

| Feature | fxTunnel | Other tools |
|---|---|---|
| Free tier | No limits on traffic or connections | Often limited by time, requests, or bandwidth |
| HTTPS certificate | Valid, automatic — no browser warnings | Same in most tools |
| Stable URL | Same URL across restarts (free tier) | Often random on free tiers |
| Inspector + Replay | From $5/mo — view all requests, resend with one click | Missing or requires separate setup |
| Custom domain | From $5/mo, up to 5 tunnels | More expensive or unavailable |
| 10+ tunnels | From $10/mo | Significantly more expensive |
| Open source | Yes — [github.com/mephistofox/fxtun.dev](https://github.com/mephistofox/fxtun.dev) | Often proprietary |

We compare all three tools in depth in [ngrok vs Cloudflare vs fxTunnel](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

## FAQ

### Why does WordPress break when I open it through a tunnel?

The root cause is that WordPress stores its URL in the database (`siteurl` and `home` options). When the tunnel URL does not match, you get redirect loops, missing stylesheets, and broken links. To fix this, set `WP_HOME` and `WP_SITEURL` in `wp-config.php` to the tunnel URL, or use the dynamic snippet from Step 4 above so it adapts automatically to whatever hostname the request comes in on.

### Can I use a tunnel to demo a WordPress site to a client?

Yes -- and it takes about a minute. Start your local WordPress, open a tunnel with `fxtunnel http 8080`, make sure `WP_HOME` and `WP_SITEURL` point to the tunnel URL, and send the link to your client. They see your theme, plugins, and content exactly as they are on your machine. When the demo is over, `Ctrl+C` shuts everything down.

### How do I test WooCommerce webhooks on localhost?

Open a tunnel with `fxtunnel http 8080`, then paste the tunnel URL as the Delivery URL in WooCommerce webhook settings. When an event fires (order created, payment completed, etc.), WooCommerce delivers the payload through the tunnel to your local handler. The Inspector (from $5/mo) lets you inspect every payload and replay requests. The [Webhook Testing with a Tunnel](/blog/webhook-testing-with-tunnel/) article goes into more detail.

### Does the tunnel URL change when I restart it?

No. fxTunnel SaaS keeps the same URL across restarts, even on the free tier. If you use a custom domain (from $5/mo), you get a fully permanent address like `dev.yoursite.com`. No need to touch `wp-config.php` or webhook settings after restarting.

### Is it safe to expose my WordPress development site through a tunnel?

For development and short demos -- yes. All traffic is encrypted via [TLS](/blog/tls-13-tunnel-security/). On your side, stick to test data, protect `wp-admin` with strong credentials, and close the tunnel when the session ends. A production WordPress site should not be served through a tunnel.
