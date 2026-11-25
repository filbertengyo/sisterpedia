import fs from "node:fs";
import path from "node:path";
import { expect, type BrowserContext, type Page, type Request } from "@playwright/test";

export const SERVER_PORT = Number(process.env.PW_SERVER_PORT ?? 18000);
export const SHOTS = path.resolve(__dirname, "../../report/figures/gui");
fs.mkdirSync(SHOTS, { recursive: true });

export async function shot(page: Page, name: string) {
  await page.screenshot({ path: path.join(SHOTS, name), fullPage: true });
}

export interface Captured {
  url: string;
  method: string;
  body: string | null;
}

/** Records every request of the context, including those made by the crypto Web Worker. */
export function capture(context: BrowserContext): Captured[] {
  const out: Captured[] = [];
  context.on("request", (r: Request) => out.push({ url: r.url(), method: r.method(), body: r.postData() }));
  return out;
}

const B64 = /^[A-Za-z0-9+/]+={0,2}$/;

// Fields allowed in each request body (docs/API_CONTRACT.md); sensitive ones must be long Base64 strings.
// `params` is public (Lattigo JSON parameters, Base64) and short, so only its encoding is checked.
const CONTRACT: Record<string, { plain: string[]; cipher: string[]; publicB64?: string[] }> = {
  "/accounts": { plain: ["user_id"], cipher: ["public_key", "relin_key", "enc_balance"], publicB64: ["params"] },
  topup: { plain: [], cipher: ["enc_amount"] },
  transactions: { plain: ["product_id", "discount_code"], cipher: ["enc_quantity", "enc_metadata"] },
};

/** Asserts the trust-boundary rules on everything sent to the server. */
export function assertOnlyCiphertext(reqs: Captured[]) {
  expect(reqs.some((r) => r.url.includes(":8001"))).toBe(false); // no local client service anymore
  const posts = reqs.filter((r) => r.method === "POST" && r.url.includes(`:${SERVER_PORT}/`));
  expect(posts.length).toBeGreaterThan(0);
  for (const r of posts) {
    const p = new URL(r.url).pathname;
    const rule = p === "/accounts" ? CONTRACT["/accounts"] : p.endsWith("/topup") ? CONTRACT.topup : CONTRACT.transactions;
    const body = JSON.parse(r.body ?? "{}") as Record<string, unknown>;
    const allowed = [...rule.plain, ...rule.cipher, ...(rule.publicB64 ?? [])];
    expect(Object.keys(body).every((k) => allowed.includes(k)), `unexpected field in ${p}: ${Object.keys(body)}`).toBe(true);
    expect("secret_key" in body).toBe(false);
    for (const k of rule.cipher) {
      const v = body[k];
      expect(typeof v, `${p}.${k} must be a Base64 string`).toBe("string");
      expect((v as string).length).toBeGreaterThan(1000);
      expect(B64.test(v as string)).toBe(true);
    }
    for (const k of rule.publicB64 ?? []) expect(B64.test(String(body[k]))).toBe(true);
    for (const v of Object.values(body)) expect(typeof v).not.toBe("number"); // no plaintext numbers at all
  }
}

export async function ready(page: Page) {
  await page.goto("/");
  await expect(page.getByTestId("wasm-status")).toContainText("siap");
}

export async function activate(page: Page, userId: string, profile = "p1") {
  await page.getByTestId("tab-activate").click();
  await page.getByTestId("activate-user").fill(userId);
  await page.getByTestId("activate-profile").selectOption(profile);
  await page.getByTestId("activate-submit").click();
  await expect(page.getByTestId("activate-status")).toContainText(userId);
  await expect(page.getByTestId("active-user")).toContainText(userId);
}

export async function checkBalance(page: Page): Promise<string> {
  await page.getByTestId("tab-balance").click();
  const panel = page.getByTestId("panel-balance");
  const before = Number(await panel.getAttribute("data-checks"));
  await page.getByTestId("balance-check").click();
  await expect(panel).toHaveAttribute("data-checks", String(before + 1));
  const v = panel.getByTestId("balance-value");
  return (await v.count()) > 0 ? ((await v.textContent()) ?? "") : "";
}

export async function loadHistory(page: Page) {
  await page.getByTestId("tab-history").click();
  const panel = page.getByTestId("panel-history");
  const before = Number(await panel.getAttribute("data-loads"));
  await page.getByTestId("history-load").click();
  await expect(panel).toHaveAttribute("data-loads", String(before + 1));
}

async function textIfPresent(page: Page, testId: string): Promise<string | null> {
  const loc = page.getByTestId(testId);
  return (await loc.count()) > 0 ? await loc.first().textContent() : null;
}

export async function topup(page: Page, amount: string) {
  await page.getByTestId("tab-topup").click();
  const prev = await textIfPresent(page, "topup-balance-raw");
  await page.getByTestId("topup-amount").fill(amount);
  await page.getByTestId("topup-submit").click();
  const v = page.getByTestId("topup-balance-raw");
  await expect(v).toBeVisible();
  if (prev !== null) await expect(v).not.toHaveText(prev);
}

export async function transact(page: Page, product: string, qty: string, discount: string) {
  await page.getByTestId("tab-transact").click();
  await page.getByTestId("transact-product").selectOption(product);
  await page.getByTestId("transact-qty").fill(qty);
  await page.getByTestId("transact-discount").selectOption(discount);
  const prev = await textIfPresent(page, "transact-status");
  await page.getByTestId("transact-submit").click();
  const st = page.getByTestId("transact-status");
  await expect(st).toBeVisible();
  if (prev !== null) await expect(st).not.toHaveText(prev);
  await expect(page.getByTestId("transact-total-value")).toBeVisible();
}
