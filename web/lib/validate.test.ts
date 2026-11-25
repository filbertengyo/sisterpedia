import { describe, expect, it } from "vitest";
import { recoverMetadata } from "./crypto/metadata";
import { describeError, messageFor } from "./errors";
import { ShopError } from "./types";
import { checkBillBound, checkTopupBound, estimateBill, parseAmount, validateAmount, validateQuantity, validateUserId } from "./validate";

describe("validateUserId", () => {
  it("accepts the contract pattern", () => {
    for (const id of ["abc", "vanessa", "user_01", "a-b-c", "x".repeat(32)]) expect(validateUserId(id).ok).toBe(true);
  });
  it("rejects other ids", () => {
    for (const id of ["ab", "Vanessa", "with space", "x".repeat(33), "a/b", ""]) expect(validateUserId(id).ok).toBe(false);
  });
});

describe("amounts", () => {
  it("parses integers and decimals with . or ,", () => {
    expect(parseAmount("100000")).toBe(100000);
    expect(parseAmount("2500.75")).toBe(2500.75);
    expect(parseAmount("2500,5")).toBe(2500.5);
    expect(Number.isNaN(parseAmount("1.234"))).toBe(true);
    expect(Number.isNaN(parseAmount("-5"))).toBe(true);
    expect(Number.isNaN(parseAmount("abc"))).toBe(true);
  });
  it("validates > 0 and the bound", () => {
    expect(validateAmount(1).ok).toBe(true);
    expect(validateAmount(0).ok).toBe(false);
    expect(validateAmount(500001).ok).toBe(false);
    expect(validateAmount(Number.NaN).ok).toBe(false);
  });
  it("quantity must be a positive integer", () => {
    expect(validateQuantity(2).ok).toBe(true);
    expect(validateQuantity(0).ok).toBe(false);
    expect(validateQuantity(1.5).ok).toBe(false);
  });
});

describe("overflow guards", () => {
  it("topup keeps the balance within the bound", () => {
    expect(checkTopupBound(400000, 100000).ok).toBe(true);
    expect(checkTopupBound(450000, 100000).ok).toBe(false);
    expect(checkTopupBound(undefined, 100000).ok).toBe(true);
  });
  it("bill subtotal and resulting balance stay within the bound", () => {
    expect(checkBillBound(12500, 2, 100000).ok).toBe(true);
    expect(checkBillBound(12500, 50, 1000000).ok).toBe(false);
    expect(checkBillBound(17500, 20, -400000).ok).toBe(false);
  });
  it("estimateBill follows t = p*q*(1-d) - k", () => {
    expect(estimateBill(12500, 2, 0.1, 500)).toBeCloseTo(22000, 9);
  });
});

describe("metadata recovery", () => {
  it("rounds approximate slots", () => {
    const m = recoverMetadata([1.0000002, 1.9999998, 3.00001]);
    expect(m).toMatchObject({ productNum: 1, quantity: 2, discountNum: 3, uncertain: false });
  });
  it("flags slots far from an integer", () => {
    expect(recoverMetadata([1.4, 2, 3]).uncertain).toBe(true);
    expect(recoverMetadata([Number.NaN, 2, 3]).uncertain).toBe(true);
  });
});

describe("errors", () => {
  it("maps codes to Indonesian messages", () => {
    const d = describeError(new ShopError("account_not_found", "account x is not activated", 404));
    expect(d.code).toBe("account_not_found");
    expect(d.message).toContain("belum diaktifkan");
    expect(d.detail).toBe("account x is not activated");
    expect(messageFor("server_unreachable")).toContain("Server");
    expect(describeError(new Error("boom")).code).toBe("internal_error");
  });
});
