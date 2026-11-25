import fs from "node:fs";
import path from "node:path";
import { expect, test } from "@playwright/test";
import { unzipSync } from "fflate";
import { activate, assertOnlyCiphertext, capture, checkBalance, loadHistory, ready, shot, topup, transact } from "./helpers";

// Scenario 1: the five services end to end (activate -> balance -> topup ->
// transact -> history) with the spec's reference example 12500 x 2, COMBO
// (10% + 500) = 22000, and a check that only ciphertext crosses the network.
test("five services with ciphertext side by side", async ({ page, context }) => {
  const reqs = capture(context);
  await ready(page);

  await activate(page, "alice", "p1");
  await expect(page.getByTestId("activate-balance-value")).toHaveText("Rp 0,00");
  await expect(page.getByTestId("activate-ct-preview")).toContainText("...");
  await shot(page, "01-activate.png");

  expect(await checkBalance(page)).toBe("Rp 0,00");
  await shot(page, "02-checkbalance.png");

  await topup(page, "100000");
  await expect(page.getByTestId("topup-balance-value")).toHaveText("Rp 100.000,00");
  await topup(page, "2500,75");
  await expect(page.getByTestId("topup-balance-value")).toHaveText("Rp 102.500,75");
  await shot(page, "03-topup.png");

  await transact(page, "P001", "2", "COMBO");
  await expect(page.getByTestId("transact-total-value")).toHaveText("Rp 22.000,00");
  await expect(page.getByTestId("transact-balance-value")).toHaveText("Rp 80.500,75");
  await expect(page.getByTestId("transact-total-ct-preview")).toContainText("...");
  await shot(page, "04-transact.png");

  await transact(page, "P004", "3", "DISC10");
  await expect(page.getByTestId("transact-total-value")).toHaveText("Rp 24.300,00");

  await loadHistory(page);
  const rows = page.getByTestId("history-row");
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText("P001");
  await expect(rows.nth(0)).toContainText("COMBO");
  await expect(rows.nth(0)).toContainText("Rp 22.000,00");
  await expect(rows.nth(1)).toContainText("P004");
  await rows.nth(0).getByRole("button", { name: "Lihat" }).click();
  await shot(page, "05-gettransact.png");

  await page.getByTestId("panel-log").scrollIntoViewIfNeeded();
  await shot(page, "06-request-log.png");

  assertOnlyCiphertext(reqs);
  // the worker's requests are captured as well (Activate, 2x Topup, 2x Transact)
  expect(reqs.filter((r) => r.method === "POST").length).toBeGreaterThanOrEqual(5);
});

// Scenario 2: two users with different keys, switching without reload, and
// the error messages (invalid id, double activation, unknown account,
// overflow guard) plus a negative balance after an over-limit purchase.
test("two users, errors and negative balance", async ({ page, context }) => {
  const reqs = capture(context);
  await ready(page);
  let navigations = 0;
  page.on("framenavigated", (f) => {
    if (f === page.mainFrame()) navigations++;
  });

  await activate(page, "bob", "p1");
  await topup(page, "10000");
  await activate(page, "carol", "p2");
  await topup(page, "250000.5");

  // switch users without reloading the page
  await page.getByTestId("user-bob").click();
  await expect(page.getByTestId("active-user")).toContainText("bob");
  expect(await checkBalance(page)).toBe("Rp 10.000,00");
  await page.getByTestId("user-carol").click();
  await expect(page.getByTestId("active-user")).toContainText("carol");
  expect(await checkBalance(page)).toBe("Rp 250.000,50");
  await shot(page, "07-two-users.png");
  expect(navigations).toBe(0);

  // U06: bill above the balance -> server still subtracts, client shows negative
  await page.getByTestId("user-bob").click();
  await transact(page, "P003", "1", "NONE");
  await expect(page.getByTestId("transact-balance-value")).toHaveText("-Rp 7.500,00");
  await expect(page.getByTestId("transact-balance-negative")).toContainText("Saldo tidak mencukupi");
  await shot(page, "08-negative-balance.png");

  // invalid user id (client-side validation)
  await page.getByTestId("tab-activate").click();
  await page.getByTestId("activate-user").fill("AB");
  await page.getByTestId("activate-submit").click();
  await expect(page.getByTestId("panel-activate")).toContainText("ID pengguna harus 3-32 karakter");
  await shot(page, "09-error-invalid-id.png");

  // double activation -> 409 account_exists
  await page.getByTestId("activate-user").fill("bob");
  await page.getByTestId("activate-profile").selectOption("p1");
  await page.getByTestId("activate-submit").click();
  await expect(page.getByTestId("error-banner").last()).toHaveAttribute("data-code", "account_exists");
  await shot(page, "10-error-account-exists.png");

  // overflow guard: topup that would exceed the safe CKKS range
  await page.getByTestId("user-carol").click();
  await page.getByTestId("tab-topup").click();
  await page.getByTestId("topup-amount").fill("300000");
  await page.getByTestId("topup-submit").click();
  await expect(page.getByTestId("error-banner").last()).toHaveAttribute("data-code", "value_out_of_range");
  await shot(page, "11-error-overflow-guard.png");

  assertOnlyCiphertext(reqs);
});

// Scenario 3 (U02 in the browser): keys survive a browser restart through
// IndexedDB, and an exported zip restores the account in a fresh browser
// profile without generating new keys. Also: the secret key never appears
// in any request, and an imported key for an unactivated id gives 404.
test("key export, browser restart and import", async ({ page, context, browser }, testInfo) => {
  const reqs = capture(context);
  await ready(page);
  await activate(page, "dave", "p1");
  await topup(page, "50000");
  await transact(page, "P002", "1", "DISC10"); // 13500
  await expect(page.getByTestId("transact-balance-value")).toHaveText("Rp 36.500,00");

  // export (warning modal first)
  await page.getByTestId("tab-keys").click();
  await page.getByTestId("keys-export").click();
  await expect(page.getByTestId("export-modal")).toContainText("secret key");
  await shot(page, "12-export-warning.png");
  const [download] = await Promise.all([page.waitForEvent("download"), page.getByTestId("export-confirm").click()]);
  const zipPath = testInfo.outputPath("dave-keys.zip");
  await download.saveAs(zipPath);
  const files = unzipSync(new Uint8Array(fs.readFileSync(zipPath)));
  expect(Object.keys(files).sort()).toEqual(["params.json", "public.key", "relin.key", "secret.key"]);
  const skB64 = Buffer.from(files["secret.key"]).toString("base64");
  for (const r of reqs) expect(r.body?.includes(skB64.slice(0, 256)) ?? false).toBe(false);
  assertOnlyCiphertext(reqs);

  // browser restart: same storage (IndexedDB) -> balance still decrypts
  const state = await context.storageState({ indexedDB: true });
  await page.close();
  const restarted = await browser.newContext({ storageState: state, viewport: { width: 1400, height: 1000 } });
  const p2 = await restarted.newPage();
  await ready(p2);
  await p2.getByTestId("user-dave").click();
  expect(await checkBalance(p2)).toBe("Rp 36.500,00");
  await restarted.close();

  // fresh browser profile (empty IndexedDB) -> import the zip
  const fresh = await browser.newContext({ viewport: { width: 1400, height: 1000 } });
  const p3 = await fresh.newPage();
  await ready(p3);
  await expect(p3.getByTestId("user-list")).toHaveCount(0);
  await p3.getByTestId("tab-keys").click();
  await p3.getByTestId("import-user").fill("dave");
  await p3.getByTestId("import-file").setInputFiles(zipPath);
  await p3.getByTestId("import-submit").click();
  await expect(p3.getByTestId("keys-message")).toContainText("berhasil diimpor");
  expect(await checkBalance(p3)).toBe("Rp 36.500,00");
  await loadHistory(p3);
  await expect(p3.getByTestId("history-row")).toHaveCount(1);
  await expect(p3.getByTestId("history-row").first()).toContainText("P002");
  await shot(p3, "13-import-restored.png");

  // the same keys imported under an id that was never activated -> 404
  await p3.getByTestId("tab-keys").click();
  await p3.getByTestId("import-user").fill("ghost");
  await p3.getByTestId("import-file").setInputFiles(zipPath);
  await p3.getByTestId("import-submit").click();
  await expect(p3.getByTestId("keys-message")).toContainText("ghost");
  await p3.getByTestId("tab-balance").click();
  await p3.getByTestId("balance-check").click();
  await expect(p3.getByTestId("error-banner").last()).toHaveAttribute("data-code", "account_not_found");
  await shot(p3, "14-error-account-not-found.png");

  // a corrupted zip is rejected
  const bad = testInfo.outputPath("bad.zip");
  fs.writeFileSync(bad, "not a zip");
  await p3.getByTestId("tab-keys").click();
  await p3.getByTestId("import-user").fill("eve");
  await p3.getByTestId("import-file").setInputFiles(bad);
  await p3.getByTestId("import-submit").click();
  await expect(p3.getByTestId("error-banner").last()).toHaveAttribute("data-code", "keys_corrupt");
  await fresh.close();
  expect(path.basename(zipPath)).toBe("dave-keys.zip");
});
