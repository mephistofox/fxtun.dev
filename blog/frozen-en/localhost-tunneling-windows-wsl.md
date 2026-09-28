---
title: "Localhost Tunneling on Windows — PowerShell & WSL"
date: 2026-03-27T14:00:00+03:00
draft: false
description: "Set up localhost tunnels on Windows: PowerShell install, WSL2 integration, Docker Desktop tunneling, firewall config, and troubleshooting."
tags: [windows, wsl, tunneling, fxTunnel, powershell, docker-desktop, developer-tools]
image: ""
author: "fxTunnel Team"
faq:
  - q: "Can I run fxTunnel natively on Windows without WSL?"
    a: "Yes. fxTunnel ships as a standalone .exe binary for Windows. Download it with PowerShell using Invoke-WebRequest or install via go install. It runs directly in PowerShell or Command Prompt with no WSL, no Docker, and no extra dependencies."
  - q: "How do I tunnel a service running inside WSL2 from Windows?"
    a: "You have two options. First, install fxTunnel inside WSL2 and run it there — this is the simplest approach because WSL2 services bind to localhost inside the Linux VM. Second, run fxTunnel on Windows and point it at the WSL2 IP address, which you can find with wsl hostname -I."
  - q: "Does Windows Firewall block fxTunnel?"
    a: "Usually no. fxTunnel makes an outbound TLS connection on port 443, which Windows Firewall allows by default. If your organization uses a third-party firewall or group policy that blocks unknown outbound connections, you may need to add an exception for fxtunnel.exe."
  - q: "Can I use fxTunnel with Docker Desktop on Windows?"
    a: "Yes. Docker Desktop on Windows maps container ports to localhost just like on Linux. Run your container with -p 8080:80, then run fxtunnel http 8080 in PowerShell. The tunnel will forward traffic to the container. You can also run fxTunnel inside a container using docker-compose."
  - q: "Why does my tunnel show connection refused when the server runs in WSL2?"
    a: "WSL2 runs a full Linux VM with its own network stack. Services that bind to 127.0.0.1 inside WSL2 may not be reachable from Windows. Either run fxTunnel inside WSL2, or make your WSL2 service bind to 0.0.0.0 and use the WSL2 IP address. Windows 11 with mirrored networking mode resolves this automatically."
---

## Why Windows Developers Need Tunnels

If you are building a .NET API in Visual Studio, running a Node.js server in WSL2, or spinning up containers in Docker Desktop, you already know the drill: at some point you need your local service reachable from the internet. Maybe it is a Stripe webhook, a demo for a remote client, an OAuth callback, or a mobile device hitting your local backend.

The tricky part on Windows is that tunneling involves several moving parts. Your service might run natively in Windows, inside a WSL2 Linux distribution, or in a Docker container. Each environment has different networking behavior, and Windows Firewall adds another layer. This guide covers every scenario: native PowerShell installation, WSL2 integration, Docker Desktop, firewall configuration, and Windows-specific troubleshooting.

For a general introduction to tunneling, see [What Is Tunneling](/blog/what-is-tunneling/).

## Installing fxTunnel on Windows (Native)

fxTunnel runs natively on Windows as a standalone `.exe` binary. No WSL or Docker required. You can install it via PowerShell in under a minute.

### Method 1: PowerShell One-Liner

Open PowerShell (regular or Administrator) and run:

```powershell
# Download and install fxTunnel for Windows
Invoke-WebRequest -Uri "https://fxtun.dev/install.ps1" -UseBasicParsing | Invoke-Expression
```

Verify the installation:

```powershell
fxtunnel --version
```

### Method 2: Go Install

If you have Go installed on Windows:

```powershell
go install github.com/mephistofox/fxtun.dev/cmd/fxtunnel@latest
```

Make sure `$env:GOPATH\bin` (typically `C:\Users\<you>\go\bin`) is in your `PATH`.

### Method 3: Manual Download

Download the Windows binary from the GitHub releases page and place it in a directory that is in your `PATH`:

```powershell
# Download the binary
Invoke-WebRequest -Uri "https://github.com/mephistofox/fxtun.dev/releases/latest/download/fxtunnel-windows-amd64.exe" -OutFile "$env:USERPROFILE\bin\fxtunnel.exe"

# Add to PATH for this session (or add permanently via System Properties)
$env:PATH += ";$env:USERPROFILE\bin"

# Verify
fxtunnel --version
```

### Your First Tunnel on Windows

Start a local web server (any framework, any port) and open a tunnel:

```powershell
# Example: a Node.js server running on port 3000
fxtunnel http 3000
```

Output:

```
fxTunnel v1.x — tunnel is active
Public URL:  https://win-demo.fxtun.dev
Forwarding:  https://win-demo.fxtun.dev -> http://localhost:3000

Press Ctrl+C to stop
```

That is it. The public URL is live, has a valid TLS certificate, and forwards requests to your local server. The [HTTPS on Localhost](/blog/https-localhost-development/) article explains how fxTunnel handles TLS automatically.

## Tunneling from WSL2

WSL2 (Windows Subsystem for Linux 2) runs a real Linux kernel inside a lightweight virtual machine. Many Windows developers use WSL2 as their primary development environment for Node.js, Python, Ruby, Go, and other Linux-native stacks. The networking model of WSL2 introduces some specifics you need to understand.

### How WSL2 Networking Works

WSL2 creates a virtual network adapter with its own IP address. Services running inside WSL2 bind to the WSL2 VM's network, not directly to the Windows host. Windows 10 and early Windows 11 builds used a NAT-based model where localhost forwarding from Windows to WSL2 was unreliable. Windows 11 (22H2+) introduced mirrored networking mode, which shares the host's network stack with WSL2 and makes localhost forwarding work seamlessly.

What this means for tunneling:

- **Mirrored mode (Windows 11 22H2+):** Services in WSL2 are reachable at `localhost` from Windows. You can run fxTunnel either on Windows or inside WSL2 — both work.
- **NAT mode (default on Windows 10):** Services in WSL2 bind to a separate IP. Running fxTunnel inside WSL2 is simpler. If you run fxTunnel on Windows, you need the WSL2 IP.

### Option A: Install fxTunnel Inside WSL2 (Recommended)

This is the simplest approach. Install fxTunnel in your WSL2 distribution and run it there — no cross-network complications.

```bash
# Inside your WSL2 terminal (Ubuntu, Debian, etc.)
curl -fsSL https://fxtun.ru/install.sh | bash

# Start a tunnel to a server running inside WSL2
fxtunnel http 8080
```

The tunnel connects directly to the service running inside the same Linux environment. No IP lookups, no firewall rules, no surprises.

### Option B: Run fxTunnel on Windows, Tunnel to WSL2

If you prefer to run fxTunnel natively on Windows and your WSL2 service is not reachable at `localhost`:

```powershell
# Step 1: Find your WSL2 IP address
wsl hostname -I
# Output example: 172.23.48.1

# Step 2: Run fxTunnel on Windows pointing at the WSL2 IP
fxtunnel http 172.23.48.1:8080
```

### Enabling Mirrored Networking (Windows 11)

If you are on Windows 11 and want the simplest experience, enable mirrored networking. Create or edit the file `%USERPROFILE%\.wslconfig`:

```powershell
# Create .wslconfig with mirrored networking
@"
[wsl2]
networkingMode=mirrored
"@ | Out-File -FilePath "$env:USERPROFILE\.wslconfig" -Encoding UTF8
```

Then restart WSL:

```powershell
wsl --shutdown
wsl
```

With mirrored mode, services inside WSL2 are directly reachable at `localhost` from Windows, and you can run fxTunnel from either side.

## Docker Desktop on Windows

Docker Desktop for Windows runs containers inside a WSL2 backend (or Hyper-V on older setups). It maps container ports to `localhost` on the Windows host, so tunneling works exactly like a native Windows service.

### Tunneling a Docker Container

```powershell
# Start a container with a port mapping
docker run -d -p 8080:80 --name my-app nginx

# Open a tunnel to the mapped port
fxtunnel http 8080
```

The container is now publicly accessible. More Docker workflows, including docker-compose integration, are covered in [Docker + Tunnel: Access Containers from the Internet](/blog/docker-tunnel-expose-containers/).

### Docker Compose with fxTunnel

```yaml
version: "3.8"

services:
  web:
    image: nginx
    ports:
      - "8080:80"

  tunnel:
    image: ghcr.io/mephistofox/fxtunnel:latest
    command: ["http", "web:80"]
    depends_on:
      - web
```

```powershell
docker compose up
```

The tunnel container reaches the web container by service name through Docker's internal DNS. The public URL appears in the logs.

## Windows Firewall Configuration

Windows Firewall is the most common source of tunneling issues on Windows. fxTunnel makes an outbound TLS connection on port 443, which is allowed by default. However, corporate environments, third-party security software, and custom group policies can block it.

### Checking if fxTunnel Can Connect

```powershell
# Test outbound connectivity to the fxTunnel server
Test-NetConnection -ComputerName fxtun.dev -Port 443
```

If `TcpTestSucceeded` shows `True`, the connection is not blocked. If it shows `False`, you need to add a firewall exception.

### Adding a Firewall Exception

```powershell
# Run PowerShell as Administrator
New-NetFirewallRule -DisplayName "fxTunnel" `
  -Direction Outbound `
  -Program "$env:USERPROFILE\bin\fxtunnel.exe" `
  -Action Allow `
  -Protocol TCP `
  -RemotePort 443
```

Adjust the `-Program` path to wherever your `fxtunnel.exe` is located.

### Windows Defender and Third-Party Antivirus

Some antivirus programs flag unknown binaries that make outbound connections. If fxTunnel is blocked or quarantined:

1. Add `fxtunnel.exe` to the antivirus exclusion list.
2. If using Windows Defender, add an exclusion via PowerShell:

```powershell
# Add exclusion for fxTunnel binary
Add-MpPreference -ExclusionPath "$env:USERPROFILE\bin\fxtunnel.exe"
```

## Tunneling Different Protocols on Windows

fxTunnel supports HTTP, TCP, and UDP on Windows with the same commands as Linux and macOS. Here are common Windows development scenarios.

### HTTP: ASP.NET / .NET API

```powershell
# Your ASP.NET Core app runs on port 5000
dotnet run --urls "http://localhost:5000"

# In another terminal
fxtunnel http 5000
```

### TCP: SQL Server, PostgreSQL

```powershell
# Tunnel to a local SQL Server instance (default port 1433)
fxtunnel tcp 1433

# Tunnel to PostgreSQL
fxtunnel tcp 5432
```

### UDP: Game Servers, VoIP

```powershell
# Tunnel a game server
fxtunnel udp 27015
```

fxTunnel supports all three protocols on the free tier, including UDP -- which most other tunneling tools lack entirely. The [TCP & UDP Tunneling Explained](/blog/tcp-udp-tunneling-explained/) article covers the differences.

## Troubleshooting Windows-Specific Issues

Something not working? Most Windows tunneling problems fall into three categories: firewall and antivirus interference, WSL2 networking quirks, and port conflicts. The table below covers the most common issues.

| Problem | Symptom | Cause | Solution |
|---|---|---|---|
| **Firewall blocks outbound** | `fxtunnel` hangs or shows connection timeout | Windows Firewall or group policy blocks unknown outbound connections | Add a firewall rule for `fxtunnel.exe` allowing outbound TCP on port 443 |
| **Antivirus quarantines binary** | `fxtunnel.exe` disappears after download or is blocked on launch | Windows Defender or third-party AV flags the binary | Add `fxtunnel.exe` to the exclusion list |
| **WSL2 service not reachable** | `connection refused` when tunneling to a WSL2 port from Windows | WSL2 uses NAT networking; service binds to WSL2 VM IP, not Windows localhost | Run fxTunnel inside WSL2, or enable mirrored networking, or use `wsl hostname -I` to get the correct IP |
| **Port already in use** | `bind: address already in use` when starting the local server | Another process occupies the port | Run `netstat -ano | findstr :8080` to find the PID, then `Stop-Process -Id <PID>` |
| **Docker Desktop port conflict** | Container starts but port mapping fails | Hyper-V or another service reserves the port range | Use `netsh int ipv4 show excludedportrange protocol=tcp` to check reserved ranges; pick a different port |
| **PowerShell execution policy** | Install script refuses to run | PowerShell blocks unsigned scripts by default | Run `Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned` |
| **WSL2 DNS resolution fails** | fxTunnel inside WSL2 cannot resolve `fxtun.dev` | WSL2 DNS configuration is broken | Add `nameserver 8.8.8.8` to `/etc/resolv.conf` inside WSL2, or set `[wsl2] dnsTunneling=true` in `.wslconfig` |
| **Slow tunnel on WSL2** | High latency or timeouts | WSL2 filesystem I/O overhead when serving files from `/mnt/c/` | Keep project files on the Linux filesystem (`~/projects/`), not on the mounted Windows drive |

### Diagnosing Port Conflicts

```powershell
# Find what is using port 8080
netstat -ano | findstr :8080

# Output example:
#   TCP    0.0.0.0:8080    0.0.0.0:0    LISTENING    12345

# Kill the process
Stop-Process -Id 12345
```

### Checking WSL2 Connectivity

```bash
# Inside WSL2: verify the service is running
curl http://localhost:8080

# From Windows PowerShell: check if WSL2 port is reachable
wsl curl http://localhost:8080
```

### Checking Hyper-V Reserved Ports

Docker Desktop and Hyper-V sometimes reserve port ranges that conflict with your application:

```powershell
# List reserved port ranges
netsh int ipv4 show excludedportrange protocol=tcp
```

If your port falls within a reserved range, choose a different port for your application.

## Advanced: Persistent Tunnels and Custom Domains

### Running fxTunnel as a Background Process

On Windows, you can run fxTunnel in the background using `Start-Process`:

```powershell
# Start fxTunnel in the background
Start-Process -NoNewWindow -FilePath "fxtunnel" -ArgumentList "http", "8080"
```

For a more robust setup, consider running fxTunnel as a Windows Service using tools like `NSSM` (Non-Sucking Service Manager) or as a scheduled task that starts at login.

### Custom Domains

The free tier assigns a random subdomain on every start. For stable URLs -- webhook endpoints, OAuth redirect URIs, shared development links -- use a custom domain:

```powershell
fxtunnel http 8080 --domain=dev.yoursite.com
```

Custom domains, the request inspector, and replay are available from $5/mo. The inspector lets you examine every incoming request in real time, which is invaluable for debugging webhooks. From $10/mo, you get 10+ simultaneous tunnels for microservice architectures.

### Using the Traffic Inspector

The built-in traffic inspector lets you view, filter, and replay HTTP requests passing through the tunnel. This is especially useful on Windows where tools like `tcpdump` are not natively available. The [Traffic Inspector: Debug Requests in Real Time](/blog/traffic-inspector-debug-requests/) article walks through the full workflow.

## Best Practices for Windows Tunneling

Follow these guidelines to avoid common pitfalls when tunneling on Windows:

- **Use WSL2 for Linux stacks.** If your application targets Linux in production, develop and tunnel from inside WSL2. This eliminates cross-platform surprises and matches your production environment.
- **Keep projects on the Linux filesystem in WSL2.** Files stored under `/mnt/c/` (the Windows drive mount) have significantly worse I/O performance. Store your code in `~/projects/` inside WSL2 for fast builds and fast tunnel responses.
- **Close tunnels when idle.** Press `Ctrl+C` when you are done. An open tunnel is an open door to your machine. This applies to all operating systems, but on Windows the tunnel process can survive terminal closure if launched in the background.
- **Use test data.** Never tunnel to a production database or use live API keys. This is a universal rule, but worth repeating given how easy fxTunnel makes it to expose any port.
- **Pick fxTunnel over SSH tunnels on Windows.** SSH tunneling on Windows requires an SSH client, a remote server, manual port forwarding, and produces no HTTPS URL. fxTunnel does everything in one command. The [SSH Tunnels vs Modern Tools](/blog/ssh-tunnel-vs-modern-tools/) comparison explains why.

## FAQ

### Can I run fxTunnel natively on Windows without WSL?

Yes -- it ships as a standalone `.exe` with no external dependencies. Grab it through PowerShell with `Invoke-WebRequest`, or use `go install` if you have the Go toolchain. It works in PowerShell and Command Prompt without needing WSL or Docker.

### How do I tunnel a service running inside WSL2 from Windows?

Two ways. The easiest is to install fxTunnel inside WSL2 and run it there, since WSL2 services bind to localhost within the Linux VM. Alternatively, run fxTunnel on Windows and point it at the WSL2 IP (find it with `wsl hostname -I`).

### Does Windows Firewall block fxTunnel?

In most cases, no. fxTunnel only makes an outbound TLS connection on port 443, which Windows Firewall allows by default. Corporate firewalls or group policies that block unknown outbound connections may require an explicit exception for `fxtunnel.exe`.

### Can I use fxTunnel with Docker Desktop on Windows?

Yes. Docker Desktop maps container ports to localhost the same way it does on Linux. Start your container with `-p 8080:80`, then run `fxtunnel http 8080` in PowerShell -- traffic will flow to the container. You can also run fxTunnel inside a container via docker-compose. The [Docker + Tunnel](/blog/docker-tunnel-expose-containers/) article covers more workflows.

### Why does my tunnel show connection refused when the server runs in WSL2?

WSL2 runs a full Linux VM with its own network stack, so services bound to `127.0.0.1` inside WSL2 are not automatically reachable from Windows. The fix: either run fxTunnel inside WSL2, or bind your WSL2 service to `0.0.0.0` and use the WSL2 IP. On Windows 11, enabling mirrored networking mode makes this a non-issue.
