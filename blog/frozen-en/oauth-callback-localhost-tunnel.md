---
title: "OAuth Callback on Localhost via Tunnel — Quick Setup"
date: 2026-03-09T14:00:00+03:00
draft: false
description: "Test OAuth callbacks on localhost with fxTunnel: Google, GitHub, Facebook setup, redirect URIs, code examples, and troubleshooting."
tags: [oauth, authentication, testing, fxTunnel, localhost, tunnel, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Why does OAuth not work on localhost?"
    a: "Most OAuth providers require an HTTPS redirect URI with a publicly reachable domain. localhost is not routable from the internet and many providers reject http://localhost as insecure. A tunnel like fxTunnel gives your localhost a public HTTPS URL that providers accept as a valid redirect URI."
  - q: "Do I need to change the redirect URI every time I restart the tunnel?"
    a: "With fxTunnel SaaS, the URL stays the same across restarts — even on the free tier. With a custom domain (from $5/mo), the URL is fully under your control. You do not need to update OAuth redirect URIs after restarting the tunnel."
  - q: "Is it safe to test OAuth through a tunnel?"
    a: "For development and testing, yes. fxTunnel encrypts all traffic via TLS 1.3. Use test OAuth credentials (not production client secrets) and test user accounts. The tunnel only exposes the port you specify — nothing else on your machine is accessible."
  - q: "Can I use a tunnel for OAuth in production?"
    a: "Tunnels are designed for development and testing. For production OAuth, deploy your application to a server with a proper domain, TLS certificate, and reverse proxy. fxTunnel with a custom domain (from $5/mo) can work for staging environments."
  - q: "Which OAuth providers require HTTPS for redirect URIs?"
    a: "Google, Facebook, Apple, and Microsoft require HTTPS redirect URIs in production. GitHub allows http://localhost for development but requires HTTPS for production. A tunnel solves this for all providers at once by providing a public HTTPS URL."
---

## The Problem: OAuth Callbacks Need a Public HTTPS URL

If you have ever tried to test "Login with Google" on localhost, you know the pain. OAuth providers require a redirect URI -- a public HTTPS URL where the provider sends the user after authentication. Your localhost does not qualify. A [local tunnel](/blog/what-is-tunneling/) fixes this in about 30 seconds: fxTunnel creates a public HTTPS URL for your localhost, and OAuth callbacks just work.

Here is the flow: your application redirects the user to the provider (Google, GitHub, Facebook), the user logs in and grants permissions, and the provider redirects back to your application with an authorization code. The catch is in that "redirect back" step -- the provider sends the user to the URL you registered in OAuth settings. If that URL is `http://localhost:3000/auth/callback`, the redirect happens in the user's browser, so technically it works. But many providers reject `http://` URLs or require domain validation, and testing with teammates or on mobile devices fails entirely.

The classic workarounds -- deploying to a staging server after every change, or relying on `localhost` exceptions that only some providers allow -- are slow, unreliable, and inconsistent. A tunnel gives every developer on the team a stable HTTPS URL that works with every OAuth provider.

## How OAuth Works — and Where Localhost Breaks

The OAuth 2.0 Authorization Code flow involves three parties: the user's browser, the OAuth provider, and your application. Understanding where the redirect URI fits helps explain why a [tunnel](/blog/expose-localhost-to-internet/) is the cleanest solution for local development.

{{< diagram >}}
User's Browser                OAuth Provider              Your App (localhost)
     |                            |                            |
     |  1. Click "Login with Google"                           |
     |  ────────────────────────────────────────────────────>  |
     |                            |                            |
     |  2. Redirect to provider   |                            |
     |  <──────────────────────── | (provider login page)      |
     |                            |                            |
     |  3. User logs in + grants  |                            |
     |  ─────────────────────────>|                            |
     |                            |                            |
     |  4. Redirect to redirect_uri with ?code=xxx             |
     |  <─────────────────────────| ────> redirect_uri         |
     |                            |                            |
     |  5. Browser navigates to redirect_uri                   |
     |  ────────────────────────────────────────────────────>  |
     |                            |                            |
     |                            |  6. App exchanges code     |
     |                            |  <──────────────────────── |
     |                            |                            |
     |                            |  7. Provider returns token |
     |                            |  ─────────────────────────>|
     |                            |                            |
{{< /diagram >}}

Step 4 is where the problem lies. The provider redirects the user's browser to the `redirect_uri`. If that URI is `https://abc123.fxtun.dev/auth/callback`, the browser makes a request to the tunnel's public URL, the [tunnel server](/blog/fxtunnel-architecture/) forwards it to your `localhost:3000`, and your application receives the authorization code. The entire flow works seamlessly.

### Why Localhost Alone Is Not Enough

Different OAuth providers have different rules for redirect URIs:

| Provider | `http://localhost` allowed? | HTTPS required? | Domain validation? |
|---|---|---|---|
| **Google** | Yes (dev only) | Production: yes | Yes (authorized domains) |
| **GitHub** | Yes | No (but recommended) | No |
| **Facebook** | No (HTTPS required) | Yes | Yes (app domains) |
| **Apple** | No | Yes | Yes (associated domains) |
| **Microsoft** | Yes (dev only) | Production: yes | Yes (redirect URIs) |
| **Twitter/X** | No | Yes | Yes (callback URL) |

Even when a provider allows `http://localhost`, there are practical problems:

- **Teammates cannot test your branch** — `localhost` only works on your machine.
- **Mobile testing is impossible** — a phone cannot reach your laptop's localhost.
- **Some libraries enforce HTTPS** — OAuth SDKs may reject non-HTTPS redirect URIs.
- **Inconsistent behavior** — what works for GitHub may not work for Google in production mode.

A tunnel eliminates all of these problems at once. One command, one HTTPS URL, every provider works.

## Setting Up OAuth Testing with fxTunnel

Getting OAuth callbacks working locally takes three steps: install the CLI, start a tunnel, and register the public HTTPS URL as your redirect URI. fxTunnel works as a SaaS -- no server setup, no DNS configuration, no certificate management.

### Step 1. Install fxTunnel

```bash
# Quick install (Linux/macOS)
curl -fsSL https://fxtun.ru/install.sh | bash

# Verify the installation
fxtunnel --version
```

### Step 2. Start Your Application

Make sure your web application is running locally. For this guide, the examples use port 3000:

```bash
# Example: start a Next.js app
npm run dev
# -> http://localhost:3000

# Example: start a Django app
python manage.py runserver 3000
# -> http://localhost:3000
```

### Step 3. Create a Tunnel

```bash
# Create an HTTPS tunnel to your local server
fxtunnel http 3000
```

The output will show the public URL:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://oauth-dev.fxtun.dev
Forwarding:  https://oauth-dev.fxtun.dev -> http://localhost:3000

Press Ctrl+C to stop
```

Now `https://oauth-dev.fxtun.dev` is your base URL. The redirect URI will be something like `https://oauth-dev.fxtun.dev/auth/callback`.

### Step 4. Register the Redirect URI

Go to the OAuth provider's developer console and add the tunnel URL as a redirect URI. Specific instructions for each provider are below.

## OAuth Setup for Popular Providers

Below are ready-to-use configurations for Google, GitHub, and Facebook OAuth with fxTunnel. Each section covers the developer console setup, the redirect URI format, and a working code example. All examples assume you have `fxtunnel http 3000` running.

### Google OAuth: Login with Google

Google OAuth is the most common authentication method on the web. Google allows `http://localhost` for testing but requires HTTPS for production redirect URIs. A tunnel lets developers test with the exact same HTTPS flow that will run in production.

#### Developer Console Setup

1. Open [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
2. Create or select a project.
3. Navigate to **Credentials** and click **Create Credentials > OAuth client ID**.
4. Application type: **Web application**.
5. Under **Authorized redirect URIs**, add: `https://oauth-dev.fxtun.dev/auth/google/callback`.
6. Copy the **Client ID** and **Client Secret**.

#### Express.js Example with Passport

```javascript
// app.js — Google OAuth with Passport.js
const express = require('express');
const passport = require('passport');
const GoogleStrategy = require('passport-google-oauth20').Strategy;
const session = require('express-session');

const app = express();

app.use(session({
  secret: 'dev-secret',
  resave: false,
  saveUninitialized: false,
}));
app.use(passport.initialize());
app.use(passport.session());

passport.use(new GoogleStrategy({
    clientID: process.env.GOOGLE_CLIENT_ID,
    clientSecret: process.env.GOOGLE_CLIENT_SECRET,
    // Use the tunnel URL as the callback
    callbackURL: 'https://oauth-dev.fxtun.dev/auth/google/callback',
  },
  (accessToken, refreshToken, profile, done) => {
    console.log('Google profile:', profile.displayName, profile.emails[0].value);
    return done(null, profile);
  }
));

passport.serializeUser((user, done) => done(null, user));
passport.deserializeUser((user, done) => done(null, user));

// Redirect to Google
app.get('/auth/google',
  passport.authenticate('google', { scope: ['profile', 'email'] })
);

// Google redirects back here
app.get('/auth/google/callback',
  passport.authenticate('google', { failureRedirect: '/login' }),
  (req, res) => {
    console.log('OAuth success! User:', req.user.displayName);
    res.redirect('/dashboard');
  }
);

app.get('/dashboard', (req, res) => {
  if (!req.user) return res.redirect('/auth/google');
  res.send(`Welcome, ${req.user.displayName}!`);
});

app.listen(3000, () => console.log('Server on port 3000'));
```

```bash
# Start the app with OAuth credentials
GOOGLE_CLIENT_ID=xxx GOOGLE_CLIENT_SECRET=yyy node app.js

# In another terminal, start the tunnel
fxtunnel http 3000
```

Open `https://oauth-dev.fxtun.dev/auth/google` in a browser. Google will show the consent screen, and after approval, the browser will redirect to `https://oauth-dev.fxtun.dev/auth/google/callback` — which the tunnel forwards to your local server.

### GitHub OAuth: Login with GitHub

GitHub OAuth is widely used for developer tools, CLI applications, and code-related platforms. GitHub is one of the most permissive providers — it allows `http://localhost` for development. However, using a tunnel provides a consistent HTTPS experience and enables testing from other devices.

#### Developer Console Setup

1. Open [GitHub Developer Settings](https://github.com/settings/developers).
2. Click **New OAuth App**.
3. Homepage URL: `https://oauth-dev.fxtun.dev`.
4. Authorization callback URL: `https://oauth-dev.fxtun.dev/auth/github/callback`.
5. Copy the **Client ID** and generate a **Client Secret**.

#### Express.js Example

```javascript
// github-oauth.js
const express = require('express');
const axios = require('axios');
const app = express();

const CLIENT_ID = process.env.GITHUB_CLIENT_ID;
const CLIENT_SECRET = process.env.GITHUB_CLIENT_SECRET;
const REDIRECT_URI = 'https://oauth-dev.fxtun.dev/auth/github/callback';

// Step 1: Redirect to GitHub
app.get('/auth/github', (req, res) => {
  const params = new URLSearchParams({
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: 'user:email',
  });
  res.redirect(`https://github.com/login/oauth/authorize?${params}`);
});

// Step 2: Handle the callback
app.get('/auth/github/callback', async (req, res) => {
  const { code } = req.query;

  if (!code) {
    return res.status(400).send('Missing authorization code');
  }

  try {
    // Exchange code for access token
    const tokenResponse = await axios.post(
      'https://github.com/login/oauth/access_token',
      {
        client_id: CLIENT_ID,
        client_secret: CLIENT_SECRET,
        code: code,
        redirect_uri: REDIRECT_URI,
      },
      { headers: { Accept: 'application/json' } }
    );

    const accessToken = tokenResponse.data.access_token;

    // Fetch user profile
    const userResponse = await axios.get('https://api.github.com/user', {
      headers: { Authorization: `Bearer ${accessToken}` },
    });

    console.log('GitHub user:', userResponse.data.login);
    res.json({
      login: userResponse.data.login,
      name: userResponse.data.name,
      email: userResponse.data.email,
    });
  } catch (error) {
    console.error('OAuth error:', error.message);
    res.status(500).send('Authentication failed');
  }
});

app.listen(3000, () => console.log('Server on port 3000'));
```

### Facebook OAuth: Login with Facebook

Facebook has strict requirements for OAuth: HTTPS is mandatory for redirect URIs, and the domain must be added to the app's settings. This makes a tunnel essential for local development — without it, Facebook OAuth cannot be tested on localhost at all.

#### Developer Console Setup

1. Open [Facebook for Developers](https://developers.facebook.com/apps/).
2. Create or select an app.
3. Add the **Facebook Login** product.
4. Under **Settings > Basic**, add `oauth-dev.fxtun.dev` to **App Domains**.
5. Under **Facebook Login > Settings**, add `https://oauth-dev.fxtun.dev/auth/facebook/callback` to **Valid OAuth Redirect URIs**.
6. Copy the **App ID** and **App Secret**.

#### Express.js Example

```javascript
// facebook-oauth.js
const express = require('express');
const axios = require('axios');
const app = express();

const APP_ID = process.env.FACEBOOK_APP_ID;
const APP_SECRET = process.env.FACEBOOK_APP_SECRET;
const REDIRECT_URI = 'https://oauth-dev.fxtun.dev/auth/facebook/callback';

app.get('/auth/facebook', (req, res) => {
  const params = new URLSearchParams({
    client_id: APP_ID,
    redirect_uri: REDIRECT_URI,
    scope: 'email,public_profile',
    response_type: 'code',
  });
  res.redirect(`https://www.facebook.com/v18.0/dialog/oauth?${params}`);
});

app.get('/auth/facebook/callback', async (req, res) => {
  const { code } = req.query;

  if (!code) {
    return res.status(400).send('Missing authorization code');
  }

  try {
    // Exchange code for access token
    const tokenResponse = await axios.get(
      'https://graph.facebook.com/v18.0/oauth/access_token', {
        params: {
          client_id: APP_ID,
          client_secret: APP_SECRET,
          redirect_uri: REDIRECT_URI,
          code: code,
        },
      }
    );

    const accessToken = tokenResponse.data.access_token;

    // Fetch user profile
    const userResponse = await axios.get('https://graph.facebook.com/me', {
      params: {
        fields: 'id,name,email',
        access_token: accessToken,
      },
    });

    console.log('Facebook user:', userResponse.data.name);
    res.json(userResponse.data);
  } catch (error) {
    console.error('OAuth error:', error.message);
    res.status(500).send('Authentication failed');
  }
});

app.listen(3000, () => console.log('Server on port 3000'));
```

## Using a Tunnel with OAuth in Different Frameworks

The fxTunnel approach works with any web framework. Below are configuration snippets for popular stacks. In every case, the only change is setting the redirect URI to the tunnel's public URL.

### Next.js with NextAuth.js

```javascript
// pages/api/auth/[...nextauth].js
import NextAuth from 'next-auth';
import GoogleProvider from 'next-auth/providers/google';
import GitHubProvider from 'next-auth/providers/github';

export default NextAuth({
  providers: [
    GoogleProvider({
      clientId: process.env.GOOGLE_CLIENT_ID,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET,
    }),
    GitHubProvider({
      clientId: process.env.GITHUB_CLIENT_ID,
      clientSecret: process.env.GITHUB_CLIENT_SECRET,
    }),
  ],
});
```

```bash
# .env.local
NEXTAUTH_URL=https://oauth-dev.fxtun.dev
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=yyy
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=yyy
```

NextAuth.js uses `NEXTAUTH_URL` to construct redirect URIs automatically. Set it to the tunnel URL and everything works.

### Django with django-allauth

```python
# settings.py
SITE_ID = 1

# Set the tunnel URL as the base
ACCOUNT_DEFAULT_HTTP_PROTOCOL = 'https'

SOCIALACCOUNT_PROVIDERS = {
    'google': {
        'APP': {
            'client_id': os.environ['GOOGLE_CLIENT_ID'],
            'secret': os.environ['GOOGLE_CLIENT_SECRET'],
        },
        'SCOPE': ['profile', 'email'],
    },
    'github': {
        'APP': {
            'client_id': os.environ['GITHUB_CLIENT_ID'],
            'secret': os.environ['GITHUB_CLIENT_SECRET'],
        },
        'SCOPE': ['user:email'],
    },
}
```

```bash
# Update the Django Site object to use the tunnel URL
python manage.py shell -c "
from django.contrib.sites.models import Site
site = Site.objects.get(id=1)
site.domain = 'oauth-dev.fxtun.dev'
site.name = 'Dev (tunnel)'
site.save()
"
```

### Ruby on Rails with OmniAuth

```ruby
# config/initializers/omniauth.rb
Rails.application.config.middleware.use OmniAuth::Builder do
  provider :google_oauth2,
    ENV['GOOGLE_CLIENT_ID'],
    ENV['GOOGLE_CLIENT_SECRET'],
    { callback_path: '/auth/google/callback' }

  provider :github,
    ENV['GITHUB_CLIENT_ID'],
    ENV['GITHUB_CLIENT_SECRET'],
    { callback_path: '/auth/github/callback' }
end
```

```bash
# Set the tunnel URL as the host in development
RAILS_HOST=oauth-dev.fxtun.dev rails server -p 3000
```

### Go with golang.org/x/oauth2

```go
package main

import (
    "fmt"
    "net/http"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

var googleOauthConfig = &oauth2.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    // Tunnel URL as redirect
    RedirectURL:  "https://oauth-dev.fxtun.dev/auth/google/callback",
    Scopes:       []string{"profile", "email"},
    Endpoint:     google.Endpoint,
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
    url := googleOauthConfig.AuthCodeURL("state-token")
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    token, err := googleOauthConfig.Exchange(r.Context(), code)
    if err != nil {
        http.Error(w, "Token exchange failed", http.StatusBadRequest)
        return
    }
    fmt.Fprintf(w, "Access token: %s", token.AccessToken[:20]+"...")
}

func main() {
    http.HandleFunc("/auth/google", handleGoogleLogin)
    http.HandleFunc("/auth/google/callback", handleGoogleCallback)
    fmt.Println("Server on port 3000")
    http.ListenAndServe(":3000", nil)
}
```

## Dynamic Redirect URI with Environment Variables

Hardcoding the tunnel URL in the application is fragile — the URL may change between sessions or developers. The best practice is to use an environment variable for the base URL and construct the redirect URI dynamically.

```javascript
// config.js — dynamic redirect URI
const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';

module.exports = {
  google: {
    callbackURL: `${BASE_URL}/auth/google/callback`,
  },
  github: {
    callbackURL: `${BASE_URL}/auth/github/callback`,
  },
  facebook: {
    callbackURL: `${BASE_URL}/auth/facebook/callback`,
  },
};
```

```bash
# Start with tunnel URL
BASE_URL=https://oauth-dev.fxtun.dev node app.js

# Start without tunnel (local-only)
node app.js
# -> uses http://localhost:3000 by default
```

This approach means the same code works both with and without a tunnel. Developers on the team set `BASE_URL` to their own tunnel URL, and the application adapts automatically.

## Troubleshooting: Common OAuth + Tunnel Issues

Most OAuth issues with a tunnel come down to three things: the redirect URI does not match, the tunnel is not running, or the provider rejects the domain. The [fxTunnel Inspector](/blog/traffic-inspector-debug-requests/) can help you see whether the callback request actually reached the tunnel.

| Problem | Possible cause | Solution |
|---|---|---|
| `redirect_uri_mismatch` error | URI in code does not match URI in provider settings | Copy the exact tunnel URL from fxTunnel output and paste it into the provider's developer console. Ensure the path matches exactly (including trailing slashes). |
| OAuth page loads but callback fails | Tunnel is not running | Make sure `fxtunnel` is running. Check that the URL is reachable by opening it in a browser. |
| `invalid_client` error | Wrong client ID or secret | Double-check the environment variables. Use the correct credentials for the development environment, not production. |
| Infinite redirect loop | Session cookie not persisting | Set the cookie domain to match the tunnel URL. For Express: `cookie: { domain: '.fxtun.dev', secure: true }`. |
| CORS errors after callback | Frontend and backend on different origins | Ensure the frontend uses the same tunnel URL as the backend, or configure CORS to allow the tunnel domain. |
| Provider rejects the domain | Domain not added to provider settings | Add `oauth-dev.fxtun.dev` to the list of authorized domains in the provider's developer console (Google, Facebook require this). |
| Token exchange fails | Backend cannot reach provider API | This is not a tunnel issue — the backend calls the provider API directly, not through the tunnel. Check your internet connection and firewall rules. |
| Different URL after tunnel restart | Random subdomain reassigned | Use a custom domain in fxTunnel (from $5/mo) for a stable URL. Or use fxTunnel SaaS, which keeps the URL consistent across restarts even on the free tier. |

## OAuth Testing Best Practices

Getting OAuth right during local development saves hours of debugging later and keeps security issues out of production. These five principles apply regardless of which [tunneling tool](/blog/what-is-tunneling/) you use.

### 1. Use Separate OAuth Apps for Development

Create a dedicated OAuth application in each provider's developer console for local development. Do not use production credentials during testing. This prevents accidental data leaks and keeps the redirect URI lists clean.

### 2. Use Environment Variables for All Secrets

Never hardcode client IDs, client secrets, or redirect URIs in the source code. Use environment variables or a `.env` file (excluded from version control via `.gitignore`).

```bash
# .env — never commit this file
GOOGLE_CLIENT_ID=123456789.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
GITHUB_CLIENT_ID=Iv1.abc123
GITHUB_CLIENT_SECRET=abc123def456
BASE_URL=https://oauth-dev.fxtun.dev
```

### 3. Test the Complete Flow

Do not just test the happy path. Verify that your application handles:

- **Denied permissions** — the user clicks "Cancel" on the consent screen.
- **Expired authorization codes** — the code has a short lifespan (usually 10 minutes).
- **Invalid state parameter** — CSRF protection must reject mismatched state tokens.
- **Missing scopes** — the user grants fewer permissions than requested.

The [Inspector + Replay](/blog/traffic-inspector-debug-requests/) feature in fxTunnel makes it easy to re-test callback requests without going through the entire OAuth flow again.

### 4. Validate the State Parameter

Always use a `state` parameter to prevent CSRF attacks. Generate a random token, store it in the session, include it in the authorization URL, and verify it when the callback arrives.

```javascript
const crypto = require('crypto');

// Generate state before redirect
app.get('/auth/google', (req, res) => {
  const state = crypto.randomBytes(16).toString('hex');
  req.session.oauthState = state;
  const url = googleOauthConfig.authorizationUrl({ state });
  res.redirect(url);
});

// Verify state on callback
app.get('/auth/google/callback', (req, res) => {
  if (req.query.state !== req.session.oauthState) {
    return res.status(403).send('Invalid state parameter');
  }
  // ... exchange code for token
});
```

### 5. Clean Up Tunnel When Done

A running tunnel exposes your local application to the internet. When OAuth testing is complete, stop the tunnel with `Ctrl+C`. For team environments, [custom domains](/blog/custom-domain-tunnel-setup/) give each developer a consistent, predictable URL.

## Why fxTunnel Works Well for OAuth Callback Testing

fxTunnel is a SaaS tunnel with a free tier, automatic HTTPS for every tunnel, and consistent URLs across restarts. For OAuth testing, that means a single `fxtunnel http 3000` command replaces the entire deployment pipeline. Here is how it compares to [other tunneling tools](/blog/ngrok-vs-cloudflare-vs-fxtunnel/).

| Feature | fxTunnel | Other tools |
|---|---|---|
| Free tier | No limits on traffic or connections | Often limited by time, requests, or connections |
| HTTPS | Automatic for every tunnel | Usually automatic, but may require configuration |
| Consistent URL | Same URL across restarts (even free tier) | Often changes on every restart (free tiers) |
| Custom domain | From $5/mo (any DNS) | More expensive or unavailable |
| Inspector + Replay | From $5/mo — view all OAuth callbacks | Missing or requires a separate tool |
| Setup | One command, 30 seconds | Often requires configuration and sign-up |
| Open source | Yes | Often proprietary |

You can find a broader comparison in the [2026 tunneling tools ranking](/blog/best-tunneling-tools-2026/).

## FAQ

### Why does OAuth not work on localhost?

OAuth providers typically need an HTTPS redirect URI on a publicly reachable domain. `localhost` is not routable from the internet, and many providers flat-out reject `http://localhost`. A [tunnel](/blog/what-is-tunneling/) like fxTunnel gives your local server a public HTTPS address that providers accept.

### Do I need to change the redirect URI every time I restart the tunnel?

Not with fxTunnel -- the URL stays the same across restarts, even on the free tier. If you use a custom domain (from $5/mo), the URL is entirely under your control, so there is nothing to update in your OAuth settings.

### Is it safe to test OAuth through a tunnel?

For development and testing, yes. Traffic is encrypted via [TLS 1.3](/blog/tls-13-tunnel-security/), and the tunnel only exposes the port you specify. Just make sure you are using test OAuth credentials and test user accounts, not production secrets.

### Can I use a tunnel for OAuth in production?

Tunnels are designed for development and testing. For production, deploy your application to a server with a proper domain, TLS certificate, and reverse proxy. fxTunnel with a custom domain (from $5/mo) can work well for staging environments, though.

### Which OAuth providers require HTTPS for redirect URIs?

Google, Facebook, Apple, and Microsoft all require HTTPS in production. GitHub allows `http://localhost` during development but requires HTTPS for production. A tunnel gives you a public HTTPS URL that satisfies every provider at once.
