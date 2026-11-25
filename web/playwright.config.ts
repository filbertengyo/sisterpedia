import os from "node:os";
import path from "node:path";
import { defineConfig, devices } from "@playwright/test";

// The GUI is tested against the real Go server (separate process, temporary
// SQLite DB) and a production build of the web app that serves crypto.wasm.
// Run `make wasm` first so web/public/crypto.wasm exists.
const SERVER_PORT = Number(process.env.PW_SERVER_PORT ?? 18000);
const WEB_PORT = Number(process.env.PW_WEB_PORT ?? 13000);
const SERVER_URL = `http://localhost:${SERVER_PORT}`;
const WEB_URL = `http://localhost:${WEB_PORT}`;
const DB = path.join(os.tmpdir(), `sisterpedia-playwright-${Date.now()}.db`);
const GO = process.env.GO_BIN ?? "go";

export default defineConfig({
  testDir: "./e2e",
  timeout: 180_000,
  expect: { timeout: 30_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"], ["html", { open: "never", outputFolder: "playwright-report" }]],
  use: {
    baseURL: WEB_URL,
    viewport: { width: 1400, height: 1000 },
    trace: "retain-on-failure",
    acceptDownloads: true,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1400, height: 1000 } } }],
  webServer: [
    {
      command: `"${GO}" run ./cmd/server`,
      cwd: "..",
      url: `${SERVER_URL}/healthz`,
      reuseExistingServer: false,
      timeout: 180_000,
      env: {
        SERVER_ADDR: `:${SERVER_PORT}`,
        DB_PATH: DB,
        CATALOG_PATH: "data/catalog.json",
        ALLOWED_PROFILES: "p1,p2",
        ALLOWED_ORIGIN: WEB_URL,
      },
      stdout: "pipe",
    },
    {
      command: `npm run build && npm run start -- -p ${WEB_PORT}`,
      url: WEB_URL,
      reuseExistingServer: false,
      timeout: 300_000,
      env: { NEXT_PUBLIC_SERVER_API: SERVER_URL, NEXT_PUBLIC_MOCK: "0", NEXT_TELEMETRY_DISABLED: "1" },
    },
  ],
});

export { SERVER_URL, WEB_URL };
