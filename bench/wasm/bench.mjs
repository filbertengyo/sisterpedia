// WASM client benchmark: runs the browser crypto bridge (web/public/crypto.wasm,
// Lattigo compiled with GOOS=js GOARCH=wasm) under Node with the same
// procedure as cmd/bench (5 warm-up runs, 30 measured runs, monotonic clock
// performance.now()) and writes raw CSVs in the same schema
// (profile,metric,operation,run,ns) to bench/results/raw.
//
// Decryption uses the ciphertexts written by `go run ./cmd/bench` (fixtures in
// the OS temp dir), so native and WASM decrypt exactly the same inputs.
//
//   make wasm && go run ./cmd/bench && node bench/wasm/bench.mjs && go run ./cmd/bench -report
import { readFile, writeFile, mkdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "..", "..");
const args = Object.fromEntries(
  process.argv.slice(2).map((a) => {
    const [k, v] = a.replace(/^--?/, "").split("=");
    return [k, v ?? "true"];
  }),
);
const WARMUP = Number(args.warmup ?? 5);
const RUNS = Number(args.runs ?? 30);
const OUT = path.resolve(repo, args.out ?? "bench/results/raw");
const FIXTURES = args.fixtures ?? path.join(os.tmpdir(), "sisterpedia-bench");

await import(pathToFileURL(path.join(repo, "web", "public", "wasm_exec.js")).href);
const go = new globalThis.Go();
const wasm = await readFile(path.join(repo, "web", "public", "crypto.wasm"));
const { instance } = await WebAssembly.instantiate(wasm, go.importObject);
void go.run(instance);
const bridge = globalThis.sisterpedia;

function call(name, ...a) {
  const r = JSON.parse(bridge[name](...a));
  if (!r.ok) throw new Error(`${name}: ${r.error.code} ${r.error.message}`);
  return r.value;
}

function repeat(f) {
  for (let i = 0; i < WARMUP; i++) f(0);
  for (let i = 1; i <= RUNS; i++) f(i);
}

async function writeCsv(profile, metric, rows) {
  await mkdir(OUT, { recursive: true });
  const lines = ["profile,metric,operation,run,ns", ...rows.map((r) => `${profile},${metric},${r.op},${r.run},${r.ns}`)];
  await writeFile(path.join(OUT, `${profile}_${metric}.csv`), lines.join("\n") + "\n");
}

const timeIt = (f) => {
  const t0 = performance.now();
  const v = f();
  return [v, Math.round((performance.now() - t0) * 1e6)];
};

const profiles = (args.profile && args.profile !== "all" ? args.profile.split(",") : call("profiles").map((p) => p.name));
for (const profile of profiles) {
  console.log(`[${profile}] WASM keygen`);
  const kg = [];
  repeat((run) => {
    const [, ns] = timeIt(() => call("genKeys", profile));
    if (run > 0) kg.push({ op: "genKeys", run, ns });
  });
  await writeCsv(profile, "keygen_wasm", kg);

  const fixturePath = path.join(FIXTURES, `${profile}.json`);
  if (!existsSync(fixturePath)) {
    console.warn(`no fixtures at ${fixturePath}; run go run ./cmd/bench first (decrypt metrics skipped)`);
  }
  const fx = existsSync(fixturePath) ? JSON.parse(await readFile(fixturePath, "utf8")) : null;
  const keys = fx ?? call("genKeys", profile);
  const params = fx ? fx.params : keys.params;
  const pk = fx ? fx.public_key : keys.public_key;
  const sk = fx ? fx.secret_key : keys.secret_key;

  console.log(`[${profile}] WASM client crypto`);
  const cr = [];
  repeat((run) => {
    const ops = [
      ["encryptScalar amount", () => call("encryptScalar", params, pk, 100000)],
      ["encryptScalar quantity", () => call("encryptScalar", params, pk, 2)],
      ["encryptVector metadata", () => call("encryptVector", params, pk, [1, 2, 3])],
    ];
    if (fx) {
      ops.push(
        ["decryptScalar balance", () => call("decryptScalar", params, sk, fx.ct_balance)],
        ["decryptScalar total", () => call("decryptScalar", params, sk, fx.ct_total)],
        ["decryptVector metadata", () => call("decryptVector", params, sk, fx.ct_metadata, 3)],
      );
    }
    for (const [op, f] of ops) {
      const [, ns] = timeIt(f);
      if (run > 0) cr.push({ op, run, ns });
    }
  });
  await writeCsv(profile, "client_wasm", cr);
}

await writeFile(
  path.join(OUT, "environment_wasm.json"),
  JSON.stringify({ node: process.version, v8: process.versions.v8, platform: `${process.platform}/${process.arch}`, go_wasm: bridge.version }, null, 2),
);
console.log(`WASM benchmark written to ${OUT}`);
process.exit(0);
