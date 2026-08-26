#!/usr/bin/env node
//
// Fails CI if the Go relay's packet-type constants (protocol/protocol.go)
// and the Cloudflare Worker relay's RELAY_PACKET_TYPES (worker.js) ever
// drift apart. This is the automated half of "one canonical protocol" —
// see docs/connection.md and understanding.md §10.1 for why the two relay
// implementations existing independently caused real behavioral drift
// before this check existed.
//
// Usage: node scripts/check_protocol_sync.js

const fs = require("fs");
const path = require("path");

const repoRoot = path.resolve(__dirname, "..");
const goFile = path.join(repoRoot, "protocol", "protocol.go");
const workerFile = path.join(repoRoot, "worker.js");

function extractGoTypes(source) {
  // Matches lines like: TypeConnectRequest = "connect_request"
  const re = /Type\w+\s*=\s*"([a-z_]+)"/g;
  const values = new Set();
  let m;
  while ((m = re.exec(source)) !== null) {
    values.add(m[1]);
  }
  return values;
}

function extractWorkerTypes(source) {
  const marker = "const RELAY_PACKET_TYPES = {";
  const start = source.indexOf(marker);
  if (start === -1) {
    throw new Error("Could not find `const RELAY_PACKET_TYPES = { ... }` in worker.js");
  }
  const end = source.indexOf("};", start);
  if (end === -1) {
    throw new Error("Could not find the closing `};` for RELAY_PACKET_TYPES in worker.js");
  }
  const block = source.slice(start, end);

  // Matches lines like: REGISTER: "register",
  const re = /:\s*"([a-z_]+)"/g;
  const values = new Set();
  let m;
  while ((m = re.exec(block)) !== null) {
    values.add(m[1]);
  }
  return values;
}

function diff(a, b) {
  return [...a].filter((v) => !b.has(v));
}

const goSource = fs.readFileSync(goFile, "utf8");
const workerSource = fs.readFileSync(workerFile, "utf8");

const goTypes = extractGoTypes(goSource);
const workerTypes = extractWorkerTypes(workerSource);

if (goTypes.size === 0) {
  console.error(`No packet type constants found in ${goFile} — extraction regex may be broken.`);
  process.exit(1);
}
if (workerTypes.size === 0) {
  console.error(`No packet type values found in RELAY_PACKET_TYPES in ${workerFile} — extraction regex may be broken.`);
  process.exit(1);
}

const onlyInGo = diff(goTypes, workerTypes);
const onlyInWorker = diff(workerTypes, goTypes);

if (onlyInGo.length > 0 || onlyInWorker.length > 0) {
  console.error("Protocol drift detected between protocol/protocol.go and worker.js's RELAY_PACKET_TYPES:");
  if (onlyInGo.length > 0) {
    console.error(`  Only in protocol/protocol.go: ${onlyInGo.join(", ")}`);
  }
  if (onlyInWorker.length > 0) {
    console.error(`  Only in worker.js:            ${onlyInWorker.join(", ")}`);
  }
  console.error("\nBoth relays must agree on the full set of packet types. Update whichever side is missing entries.");
  process.exit(1);
}

console.log(`Protocol sync OK — ${goTypes.size} packet types match between protocol/protocol.go and worker.js.`);
