#!/bin/bash
# fxTunnel performance benchmark script
# Compares latency and throughput: direct vs tunnel
#
# Prerequisites: curl, server running, client connected with HTTP tunnel
#
# Usage:
#   ./scripts/bench.sh <direct_url> <tunnel_url> [requests]
#
# Example:
#   ./scripts/bench.sh http://localhost:3000 http://myapp.tunnel.example.com 100

set -euo pipefail

DIRECT_URL="${1:?Usage: $0 <direct_url> <tunnel_url> [requests]}"
TUNNEL_URL="${2:?Usage: $0 <direct_url> <tunnel_url> [requests]}"
REQUESTS="${3:-50}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  fxTunnel Performance Benchmark${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo ""
echo -e "Direct:  ${GREEN}${DIRECT_URL}${NC}"
echo -e "Tunnel:  ${YELLOW}${TUNNEL_URL}${NC}"
echo -e "Requests: ${REQUESTS}"
echo ""

# --- Latency test ---
echo -e "${CYAN}── Latency Test (TTFB) ──${NC}"

run_latency_test() {
    local url="$1"
    local label="$2"
    local total=0
    local min=999999
    local max=0
    local count=0

    for ((i=1; i<=REQUESTS; i++)); do
        # time_starttransfer = TTFB (Time To First Byte)
        ttfb=$(curl -o /dev/null -s -w '%{time_starttransfer}' "$url" 2>/dev/null)
        ttfb_us=$(echo "$ttfb * 1000000" | bc | cut -d. -f1)

        total=$((total + ttfb_us))
        count=$((count + 1))

        if [ "$ttfb_us" -lt "$min" ]; then min=$ttfb_us; fi
        if [ "$ttfb_us" -gt "$max" ]; then max=$ttfb_us; fi
    done

    avg=$((total / count))
    echo -e "${label}:"
    printf "  avg: %'d µs  |  min: %'d µs  |  max: %'d µs\n" "$avg" "$min" "$max"
    echo "$avg"
}

echo ""
direct_lat=$(run_latency_test "$DIRECT_URL" "  Direct " 2>&1 | tee /dev/stderr | tail -1)
tunnel_lat=$(run_latency_test "$TUNNEL_URL" "  Tunnel " 2>&1 | tee /dev/stderr | tail -1)

if [ "$direct_lat" -gt 0 ] 2>/dev/null; then
    overhead=$(( (tunnel_lat - direct_lat) ))
    ratio=$(echo "scale=1; $tunnel_lat / $direct_lat" | bc)
    echo ""
    echo -e "  ${RED}Overhead: ${overhead} µs (${ratio}x)${NC}"
fi

# --- Throughput test (1MB payload) ---
echo ""
echo -e "${CYAN}── Throughput Test (1MB download) ──${NC}"

run_throughput_test() {
    local url="$1"
    local label="$2"
    local total_bytes=0
    local total_time=0
    local runs=10

    for ((i=1; i<=runs; i++)); do
        result=$(curl -o /dev/null -s -w '%{size_download} %{time_total}' "${url}" 2>/dev/null)
        bytes=$(echo "$result" | awk '{print $1}')
        time_s=$(echo "$result" | awk '{print $2}')

        total_bytes=$((total_bytes + bytes))
        total_time=$(echo "$total_time + $time_s" | bc)
    done

    if [ "$(echo "$total_time > 0" | bc)" -eq 1 ]; then
        speed=$(echo "scale=2; $total_bytes / $total_time / 1048576" | bc)
        avg_time=$(echo "scale=3; $total_time / $runs" | bc)
        echo -e "${label}: ${speed} MB/s  (avg ${avg_time}s per request)"
    else
        echo -e "${label}: N/A (no data transferred)"
    fi
}

echo ""
run_throughput_test "$DIRECT_URL" "  Direct "
run_throughput_test "$TUNNEL_URL" "  Tunnel "

# --- Connection establishment time ---
echo ""
echo -e "${CYAN}── Connection Time ──${NC}"

run_connect_test() {
    local url="$1"
    local label="$2"
    local total=0

    for ((i=1; i<=REQUESTS; i++)); do
        connect=$(curl -o /dev/null -s -w '%{time_connect}' "$url" 2>/dev/null)
        connect_us=$(echo "$connect * 1000000" | bc | cut -d. -f1)
        total=$((total + connect_us))
    done

    avg=$((total / REQUESTS))
    printf "${label}: avg %'d µs\n" "$avg"
}

run_connect_test "$DIRECT_URL" "  Direct "
run_connect_test "$TUNNEL_URL" "  Tunnel "

echo ""
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  Done${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════${NC}"
