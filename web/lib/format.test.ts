import { describe, expect, it } from "vitest";
import { b64BinaryLength, formatBytes, formatRaw, formatRupiah, round2, truncateB64 } from "./format";

describe("formatRupiah", () => {
  it("formats with two decimals and Indonesian separators", () => {
    expect(formatRupiah(78000)).toBe("Rp 78.000,00");
    expect(formatRupiah(2500.75)).toBe("Rp 2.500,75");
  });
  it("rounds raw CKKS values for display", () => {
    expect(formatRupiah(77999.99998)).toBe("Rp 78.000,00");
  });
  it("shows negative balances and normalizes -0", () => {
    expect(formatRupiah(-12000)).toBe("-Rp 12.000,00");
    expect(formatRupiah(-0.000004)).toBe("Rp 0,00");
  });
  it("handles non-finite", () => {
    expect(formatRupiah(Number.NaN)).toBe("—");
  });
});

describe("truncateB64", () => {
  it("keeps the first 60 characters and appends ...", () => {
    const s = "A".repeat(100);
    const t = truncateB64(s);
    expect(t).toHaveLength(63);
    expect(t.endsWith("...")).toBe(true);
  });
  it("leaves short strings untouched", () => {
    expect(truncateB64("QUJD")).toBe("QUJD");
  });
});

describe("formatBytes", () => {
  it("uses binary units", () => {
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(393566)).toBe("384.3 KiB");
    expect(formatBytes(5 * 1024 * 1024)).toBe("5.00 MiB");
    expect(formatBytes(-1)).toBe("—");
  });
});

describe("helpers", () => {
  it("b64BinaryLength matches padding", () => {
    expect(b64BinaryLength("QUJD")).toBe(3);
    expect(b64BinaryLength("QUI=")).toBe(2);
    expect(b64BinaryLength("QQ==")).toBe(1);
  });
  it("round2 and formatRaw", () => {
    expect(round2(21999.999913)).toBe(22000);
    expect(round2(-0.001)).toBe(0);
    expect(formatRaw(22000.0000851)).toBe("22000.0000851000");
  });
});
