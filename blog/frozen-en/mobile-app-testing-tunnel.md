---
title: "Testing Mobile Apps Through a Tunnel on Real Devices"
date: 2026-02-27T14:00:00+03:00
draft: false
description: "How to test a mobile app on a real phone against your localhost API: fxTunnel setup for iOS, Android, React Native, and Flutter with code examples."
tags: [mobile, testing, api, fxTunnel, ios, android, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Why can't my mobile app connect to localhost?"
    a: "Because localhost (127.0.0.1) is an address that only exists on the machine itself. A phone is a separate device with its own network stack — a request to 127.0.0.1 from the phone hits the phone's own loopback, not your computer. To let the phone reach your local API, you need a public URL, which fxTunnel creates."
  - q: "Can I test on a real device without a tunnel?"
    a: "You can if the phone and computer are on the same Wi-Fi network — use the computer's local IP (e.g. 192.168.1.42:8080). But this does not work outside the office, requires manual setup, and gives you no HTTPS. fxTunnel solves all three problems: a public HTTPS URL reachable from any network."
  - q: "Does iOS block HTTP connections without HTTPS?"
    a: "Yes. App Transport Security (ATS) in iOS blocks all HTTP requests without TLS by default. fxTunnel automatically provides an HTTPS address with a valid certificate, so ATS is not an issue — no exceptions in Info.plist are needed."
  - q: "How can I see the requests my mobile app sends through the tunnel?"
    a: "Use the built-in Inspector in fxTunnel (from $5/mo): it shows all HTTP requests in real time — headers, body, and response status. This is more convenient than Charles Proxy or mitmproxy because you do not need to configure a proxy on the phone or install certificates."
  - q: "Do I need to change the URL in my app every time I restart the tunnel?"
    a: "No. With fxTunnel SaaS, the URL stays the same across restarts — even on the free tier. With a custom domain (from $5/mo), the address is fully under your control. Put the URL in a config file, and switching between localhost and tunnel takes one line."
---

## The Problem: Real Devices Cannot Reach Localhost

When building a mobile app, your API server typically runs on `localhost:8080`. The iOS Simulator or Android Emulator can reach it directly, but a real phone cannot. `localhost` resolves to the device itself, not to your computer. To test on a real iPhone or Android device, you need a public URL that points to your local server. This is exactly the problem that [tunneling](/blog/what-is-tunneling/) solves.

A typical scenario: you are building an iOS app with a backend in Go, Python, or Node.js. Everything works in the simulator, but on a real iPhone the app returns `Could not connect to the server`. The reason is simple — the phone has no idea what `localhost:8080` on your Mac means. It sends the request to its own loopback address and finds nothing there.

The classic workaround is to use your computer's local IP address (`192.168.1.42:8080`). But this approach has three problems:

- **No HTTPS** — iOS App Transport Security blocks HTTP requests, forcing you to add exceptions to `Info.plist`.
- **Network-dependent** — only works when the phone and computer share the same Wi-Fi network. At a coffee shop, at home, or on cellular data, it fails.
- **Manual configuration** — the IP address changes every time you switch networks, requiring manual updates.

## Why Emulators Are Not Enough

An emulator is a convenient tool for day-to-day development, but it does not replace testing on a real device. Critical differences show up in performance, hardware access, and OS behavior — none of which can be faithfully reproduced in a simulator.

- **Performance** — emulators run on a powerful computer and do not reflect real-world lag, FPS drops, or memory pressure on a budget Android phone.
- **Gestures and touch** — multitouch, swipes, 3D Touch / Haptic Touch, and navigation gestures behave differently on a physical screen.
- **Camera and sensors** — QR code scanning, ARKit/ARCore, accelerometer, gyroscope, Face ID / Touch ID only work on hardware.
- **Push notifications** — APNs (Apple Push Notification service) does not work in the iOS Simulator. You need a real iPhone to test pushes.
- **Network conditions** — a real device operates over Wi-Fi or cellular with latency and packet loss. An emulator uses the host machine's network directly.

For thorough testing you need three things: a local API, a real device, and a way to connect them. That third piece is where a tunnel comes in.

## The Solution: fxTunnel Gives Your API a Public HTTPS URL

fxTunnel creates a [public URL for your localhost](/blog/expose-localhost-to-internet/) with a single command. The address uses HTTPS with a valid certificate — this is critical for iOS, where App Transport Security blocks insecure connections. The phone sends requests to `https://my-api.fxtun.dev`, they travel through the [tunnel server](/blog/fxtunnel-architecture/), and arrive at your `localhost:8080`.

### Quick Start

```bash
# Install fxTunnel
curl -fsSL https://fxtun.ru/install.sh | bash

# Start your API server (Go example)
go run main.go
# -> API listening on :8080

# Create a tunnel
fxtunnel http 8080
```

You will see the public URL:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://my-api.fxtun.dev
Forwarding:  https://my-api.fxtun.dev -> http://localhost:8080
```

Now replace `http://localhost:8080` with `https://my-api.fxtun.dev` in your mobile app's configuration, and the real device can reach your API.

## Setup for iOS (Xcode + Local API + Tunnel)

When building an iOS app in Xcode, the API base URL is typically stored in a configuration file or `Info.plist`. To switch between localhost and the tunnel, extract the URL into a compile-time variable or a configuration file.

### API Client Configuration in Swift

```swift
// APIConfig.swift
import Foundation

enum APIConfig {
    // Switch between localhost (simulator) and tunnel (real device)
    #if targetEnvironment(simulator)
    static let baseURL = "http://localhost:8080"
    #else
    static let baseURL = "https://my-api.fxtun.dev"
    #endif
}

// Usage
class APIClient {
    func fetchUsers() async throws -> [User] {
        let url = URL(string: "\(APIConfig.baseURL)/api/users")!
        let (data, _) = try await URLSession.shared.data(from: url)
        return try JSONDecoder().decode([User].self, from: data)
    }
}
```

### Why HTTPS Matters for iOS

App Transport Security (ATS) in iOS blocks all HTTP requests without TLS by default. Developers often add exceptions to `Info.plist` to work around this during testing, but that is a bad practice that can accidentally ship to production.

fxTunnel automatically provides an HTTPS address with a valid TLS certificate. No ATS exceptions are needed — the app works with the tunnel the same way it works with a production server. [HTTPS on Localhost for Development](/blog/https-localhost-development/) goes deeper on this topic.

## Setup for Android (Android Studio + Local API + Tunnel)

Android development has a similar situation: the emulator can reach `10.0.2.2` as a substitute for `localhost`, but a real device needs an actual network address.

### API Client Configuration in Kotlin

```kotlin
// ApiConfig.kt
object ApiConfig {
    // Emulator: 10.0.2.2 is the host machine
    // Real device: tunnel URL
    val BASE_URL: String
        get() = if (BuildConfig.DEBUG && isEmulator()) {
            "http://10.0.2.2:8080"
        } else {
            "https://my-api.fxtun.dev"
        }

    private fun isEmulator(): Boolean {
        return (android.os.Build.FINGERPRINT.contains("generic")
                || android.os.Build.FINGERPRINT.contains("emulator"))
    }
}

// Retrofit setup
val retrofit = Retrofit.Builder()
    .baseUrl(ApiConfig.BASE_URL)
    .addConverterFactory(GsonConverterFactory.create())
    .build()
```

### Network Security Config

Starting with Android 9 (API 28), cleartext HTTP traffic is blocked by default. When using fxTunnel, this is not a problem — the tunnel provides HTTPS. But if you use a local IP instead, you need `network_security_config.xml`:

```xml
<!-- res/xml/network_security_config.xml -->
<!-- NOT needed when using fxTunnel (HTTPS out of the box) -->
<network-security-config>
    <domain-config cleartextTrafficPermitted="true">
        <domain includeSubdomains="true">10.0.2.2</domain>
    </domain-config>
</network-security-config>
```

With fxTunnel, this file is unnecessary — HTTPS works automatically.

## Setup for React Native and Flutter

Cross-platform frameworks simplify API URL management: a single line change is enough. But the underlying problem of reaching localhost from a real device remains.

### React Native

```javascript
// config/api.js
import { Platform } from 'react-native';
import DeviceInfo from 'react-native-device-info';

const getBaseUrl = () => {
  if (__DEV__) {
    // Emulator: use localhost / 10.0.2.2
    if (DeviceInfo.isEmulatorSync()) {
      return Platform.OS === 'android'
        ? 'http://10.0.2.2:8080'
        : 'http://localhost:8080';
    }
    // Real device: use tunnel
    return 'https://my-api.fxtun.dev';
  }
  // Production
  return 'https://api.myapp.com';
};

export const API_BASE_URL = getBaseUrl();

// Usage with fetch
const response = await fetch(`${API_BASE_URL}/api/users`);
```

### Flutter (Dart)

```dart
// lib/config/api_config.dart
import 'dart:io';

class ApiConfig {
  static String get baseUrl {
    // In debug mode, check whether the app runs on an emulator
    // For a real device, use the tunnel
    const bool isRelease = bool.fromEnvironment('dart.vm.product');
    if (isRelease) {
      return 'https://api.myapp.com';
    }

    // Real device: tunnel URL
    // Emulator: localhost
    // Controlled via --dart-define=API_URL=...
    return const String.fromEnvironment(
      'API_URL',
      defaultValue: 'https://my-api.fxtun.dev',
    );
  }
}

// Launch with different URLs:
// flutter run --dart-define=API_URL=http://localhost:8080      (emulator)
// flutter run --dart-define=API_URL=https://my-api.fxtun.dev   (device)
```

```bash
# Run Flutter app on a real device with the tunnel URL
flutter run --dart-define=API_URL=https://my-api.fxtun.dev
```

## Testing Push Notifications Through a Tunnel

Push notifications are one of the strongest arguments for real-device testing. APNs (Apple Push Notification service) and FCM (Firebase Cloud Messaging) do not work in simulators. For a full end-to-end test — sending a request to your API, generating a push on the server, delivering it to the device — the API must be reachable from the outside.

### Typical Flow

```
Mobile app -> POST /api/action -> your API (via tunnel)
       |
Your API builds a push notification
       |
APNs / FCM -> push to the real device
```

Without a tunnel, you would have to deploy the API to a server after every change. With fxTunnel, you test the entire cycle locally:

```bash
# API running on localhost:8080
# Tunnel is open
fxtunnel http 8080

# Test sending a push notification through the API
curl -X POST https://my-api.fxtun.dev/api/send-push \
  -H "Content-Type: application/json" \
  -d '{"user_id": "123", "title": "Test", "body": "Push via tunnel"}'
```

Your local server receives the request, sends a push through APNs or FCM, and seconds later the notification arrives on the real phone — the complete cycle without a deploy.

## Debugging: Inspector Shows Requests From Your Mobile App

Ever stared at a network error on a phone with no idea what the app actually sent? Charles Proxy and mitmproxy can help, but they require proxy configuration on the phone and root certificate installation. fxTunnel's built-in Inspector (from $5/mo) takes a different approach: it shows every request passing through the tunnel in a web interface, updating in real time.

What Inspector shows:

- **URL and method** — `POST /api/users`, `GET /api/feed`
- **Request headers** — `Authorization`, `Content-Type`, `User-Agent`
- **Request body** — the JSON payload the app sends
- **Response status** — `200 OK`, `401 Unauthorized`, `500 Internal Server Error`
- **Response body** — the JSON your API returns
- **Response time** — how many milliseconds the request took

### Replay for Resending Requests

Found a bug? No need to reproduce the scenario on the phone again. Click Replay in Inspector and the request will be resent to your API. Set a breakpoint in your IDE and step through the handling logic. This is especially valuable for complex scenarios that require navigating through a multi-step flow in the app.

The Replay technique also applies to webhook debugging -- [Webhook Testing with a Tunnel](/blog/webhook-testing-with-tunnel/) covers that workflow in detail.

## Tips: Certificate Pinning and URL Management

### Certificate Pinning

If your app uses SSL certificate pinning, requests through the tunnel will be blocked because the fxTunnel certificate will not match the pinned one. The solution:

```swift
// iOS: disable pinning in debug builds
#if DEBUG
// Use the default URLSession without a custom delegate
let session = URLSession.shared
#else
// Production: enable pinning
let session = URLSession(configuration: .default, delegate: PinningDelegate(), delegateQueue: nil)
#endif
```

```kotlin
// Android: disable pinning in debug builds
val client = if (BuildConfig.DEBUG) {
    OkHttpClient.Builder().build()  // no CertificatePinner
} else {
    OkHttpClient.Builder()
        .certificatePinner(
            CertificatePinner.Builder()
                .add("api.myapp.com", "sha256/AAAA...")
                .build()
        )
        .build()
}
```

### Managing the API URL Through Configuration

Do not hardcode the URL in your app. Extract it into configuration so switching environments takes a single line:

```bash
# .env.development
API_URL=https://my-api.fxtun.dev

# .env.production
API_URL=https://api.myapp.com
```

Running your API server in Docker? That works too -- [Docker + Tunnel](/blog/docker-tunnel-expose-containers/) walks through the setup.

## Full Workflow: From Code to Testing on the Phone

Here is the step-by-step process for developing a mobile app with fxTunnel:

1. **Start your API server** on localhost.
2. **Open a tunnel** — `fxtunnel http 8080`.
3. **Set the tunnel URL** in your mobile app configuration.
4. **Launch the app** on a real device via Xcode or Android Studio.
5. **Test** — all requests go through the tunnel to your localhost.
6. **Debug** — use Inspector to view requests and Replay to resend them.
7. **Change code** — API changes take effect instantly, with no redeploy.

```bash
# Terminal 1: API server
cd ~/projects/my-api && go run main.go

# Terminal 2: tunnel
fxtunnel http 8080

# Terminal 3: launch on device (Flutter)
cd ~/projects/my-app && flutter run --dart-define=API_URL=https://my-api.fxtun.dev
```

## Why fxTunnel Works Well for Mobile Development

fxTunnel addresses the key friction points of mobile testing: HTTPS out of the box (for iOS ATS and Android Network Security), a stable URL (no need to update config), and Inspector for debugging requests without a proxy on the phone.

| Task | Without a tunnel | With fxTunnel |
|---|---|---|
| Reaching the API from a phone | Same Wi-Fi network only | From any network |
| HTTPS for iOS ATS | Exceptions in Info.plist | Automatic |
| Viewing requests | Charles Proxy + certificates | Inspector in the browser |
| Changing API code | Redeploy to a server | Instant, no deploy |
| Push notifications | Deploy for every test | Full cycle locally |
| IP changes on network switch | Update configuration | Stable URL |

## FAQ

### Why can't my mobile app connect to localhost?

The address 127.0.0.1 (localhost) only points to the machine it runs on. When a phone sends a request to 127.0.0.1, it hits the phone's own loopback interface, not your computer. To bridge that gap you need a public URL that routes to your machine -- and that is exactly what a [tunnel](/blog/expose-localhost-to-internet/) provides.

### Can I test on a real device without a tunnel?

If both the phone and your computer share the same Wi-Fi, you can use the computer's local IP (e.g. `192.168.1.42:8080`). The downsides: it breaks outside that network, the IP changes when you switch Wi-Fi, and there is no HTTPS. A tunnel gives you a stable [HTTPS](/blog/https-localhost-development/) URL reachable from anywhere.

### Does iOS block HTTP connections without HTTPS?

It does. App Transport Security (ATS) rejects plain HTTP by default. Since fxTunnel gives you an HTTPS URL with a valid certificate, ATS passes the request through without complaints -- no need to add exceptions in `Info.plist`.

### How can I see the requests my mobile app sends through the tunnel?

Open the Inspector web UI (available from $5/mo). It lists every HTTP request in real time with full headers, body, and response status. Unlike Charles Proxy or mitmproxy, you do not need to set up a proxy on the phone or install any certificates.

### Do I need to change the URL in my app every time I restart the tunnel?

No. fxTunnel SaaS keeps the URL stable across restarts, even on the free plan. With a custom domain (from $5/mo) you choose the address yourself. Store the URL in a config file and switching between localhost and tunnel is a one-line change.
