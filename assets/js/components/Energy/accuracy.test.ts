import { describe, expect, test } from "vite-plus/test";
import { pairBuckets, predictionAccuracy } from "./accuracy";

describe("predictionAccuracy", () => {
  test("misses in both directions add up instead of cancelling out", () => {
    // 1 above, 1 below: the totals match, the buckets do not
    expect(predictionAccuracy([[1, 1], [2, 1], null, [1, 2]])).toEqual({ accuracy: 0.5, delta: 0 });
  });

  test("buckets without a prediction are left out", () => {
    const res = predictionAccuracy([[2, 1.5], null, null]);
    expect(res?.accuracy).toBeCloseTo(1 - 0.5 / 1.5);
    expect(res?.delta).toBe(0.5);
  });

  test("the error counts against what was consumed, low and high alike", () => {
    // off by 1 kWh of 2 consumed either way
    expect(predictionAccuracy([[1, 2]])).toEqual({ accuracy: 0.5, delta: -1 });
    expect(predictionAccuracy([[3, 2]])).toEqual({ accuracy: 0.5, delta: 1 });
  });

  test("no prediction or nothing consumed, no accuracy", () => {
    expect(predictionAccuracy([null, null])).toBeNull();
    expect(predictionAccuracy([[0, 3]])).toBeNull();
    expect(predictionAccuracy([[2, 0]])).toBeNull();
  });
});

describe("pairBuckets", () => {
  test("buckets without a recorded actual are left out, not counted as zero", () => {
    // bucket 1: prediction, nothing recorded; bucket 2: recorded zero; bucket 3: no prediction
    const pairs = pairBuckets([1, 2, 1, 0], [1, 0, 0, 3], new Set([0, 1, 2]), new Set([0, 2, 3]));
    expect(pairs).toEqual([[1, 1], null, [1, 0], null]);
    expect(predictionAccuracy(pairs)).toEqual({ accuracy: 0, delta: 1 });
  });
});
