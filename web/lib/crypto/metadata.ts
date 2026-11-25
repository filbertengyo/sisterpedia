// Recovery of the numeric transaction metadata [product_num, quantity,
// discount_num] from approximate CKKS values.

export const METADATA_TOLERANCE = 0.25;

export interface RecoveredMetadata {
  productNum: number;
  quantity: number;
  discountNum: number;
  uncertain: boolean;
  maxDistance: number;
}

/** Rounds each slot to the nearest integer; uncertain if any slot is more than 0.25 away. */
export function recoverMetadata(raw: number[], tolerance = METADATA_TOLERANCE): RecoveredMetadata {
  const slots = [0, 1, 2].map((i) => (Number.isFinite(raw[i]) ? raw[i] : Number.NaN));
  const r = slots.map((v) => Math.round(v));
  const distances = slots.map((v, i) => Math.abs(v - r[i]));
  const maxDistance = Math.max(...distances);
  return {
    productNum: r[0],
    quantity: r[1],
    discountNum: r[2],
    uncertain: !(maxDistance <= tolerance),
    maxDistance,
  };
}
