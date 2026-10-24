import { describe, expect, test } from "vite-plus/test";
import { pairBuckets, predictionAccuracy, splitAccuracy } from "./accuracy";

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

describe("splitAccuracy", () => {
  const sameDay = () => 0;

  test("without parts it scores like predictionAccuracy", () => {
    const total: ([number, number] | null)[] = [[1, 1], [2, 1], null, [1, 2]];
    expect(splitAccuracy(total, [], sameDay)).toEqual(predictionAccuracy(total));
  });

  test("a part's timing does not count, only its energy per group", () => {
    // 1 kWh base each bucket forecast right; the heat pump's 2 kWh forecast spread evenly,
    // run in the last bucket
    const total: [number, number][] = [
      [2, 1],
      [2, 1],
      [2, 3],
    ];
    const heatpump: [number, number][] = [
      [1, 0],
      [1, 0],
      [1, 2],
    ];
    expect(predictionAccuracy(total)?.accuracy).toBeCloseTo(1 - 3 / 5);
    // heat pump forecast 3 kWh, ran 2: off by 1 for the day, the base exact
    expect(splitAccuracy(total, [heatpump], sameDay)).toEqual({ accuracy: 1 - 1 / 5, delta: 1 });
  });

  test("each group scores a part's energy on its own", () => {
    const total: [number, number][] = [
      [2, 3],
      [2, 1],
    ];
    const heatpump: [number, number][] = [
      [1, 2],
      [1, 0],
    ];
    // per bucket groups (month view: days): the part is off by 1 in each
    expect(splitAccuracy(total, [heatpump], (i) => i)?.accuracy).toBeCloseTo(1 - 2 / 4);
    // one group (day view): the part's energy matches
    expect(splitAccuracy(total, [heatpump], sameDay)?.accuracy).toBe(1);
  });

  test("a bucket without the part's pair keeps the part in the rest", () => {
    const total: [number, number][] = [
      [2, 3],
      [2, 1],
    ];
    const heatpump: ([number, number] | null)[] = [null, [1, 0]];
    // bucket 0 scored whole (off by 1), bucket 1 base exact, part off by 1
    expect(splitAccuracy(total, [heatpump], sameDay)?.accuracy).toBeCloseTo(1 - 2 / 4);
  });
});
