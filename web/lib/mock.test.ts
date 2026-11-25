import { describe, expect, it } from "vitest";
import { createMockShop } from "./mock";

// The GUI can run fully on mock data (NEXT_PUBLIC_MOCK=1): the mock follows
// the same contract as the real shop module.
describe("mock shop", () => {
  it("runs the five services", async () => {
    const shop = createMockShop();
    const a = await shop.activate("alice", "p1");
    expect(a.status).toBe("activated");
    expect(a.balance).toBe(0);
    expect(a.enc_balance.preview.endsWith("...")).toBe(true);
    expect((await shop.topup("alice", 100000)).balance).toBe(100000);
    const t = await shop.transact("alice", "P001", 2, "COMBO");
    expect(t.total).toBe(22000);
    expect(t.balance).toBe(78000);
    expect((await shop.getBalance("alice")).balance).toBe(78000);
    const h = await shop.getHistory("alice");
    expect(h.items).toHaveLength(1);
    expect(h.items[0]).toMatchObject({ product_id: "P001", quantity: 2, discount_code: "COMBO", total: 22000 });
    expect((await shop.listUsers()).map((u) => u.user_id)).toEqual(["alice"]);
  });

  it("reports contract errors", async () => {
    const shop = createMockShop();
    await expect(shop.activate("AB", "p1")).rejects.toMatchObject({ code: "invalid_input" });
    await shop.activate("bob", "p1");
    await expect(shop.activate("bob", "p1")).rejects.toMatchObject({ code: "account_exists" });
    await expect(shop.getBalance("nobody")).rejects.toMatchObject({ code: "keys_not_found" });
    await expect(shop.transact("bob", "NOPE", 1)).rejects.toMatchObject({ code: "product_not_found" });
    await expect(shop.topup("bob", 600000)).rejects.toMatchObject({ code: "invalid_input" });
    const t = await shop.transact("bob", "P003", 1, "NONE");
    expect(t.negative).toBe(true);
    expect(t.warnings).toContain("insufficient_balance");
  });
});
