import { describe, expect, test } from "vite-plus/test";
import { priceRange } from "./subPanel";

describe("priceRange", () => {
  test("widens to round steps", () => {
    expect(priceRange(0.079, 0.389)).toEqual({ min: 0, max: 0.4, step: 0.1 });
    expect(priceRange(0.2134, 0.3378)).toEqual({ min: 0.2, max: 0.35, step: 0.05 });
  });

  test("ends already on a step stay", () => {
    expect(priceRange(0.25, 0.35)).toEqual({ min: 0.25, max: 0.35, step: 0.05 });
  });

  test("negative prices", () => {
    expect(priceRange(-0.05, 0.6)).toEqual({ min: -0.2, max: 0.6, step: 0.2 });
  });

  test("small spread gets a fine step", () => {
    expect(priceRange(0.3, 0.32)).toEqual({ min: 0.3, max: 0.32, step: 0.005 });
  });

  test("currencies priced in whole units", () => {
    expect(priceRange(0.6, 2.4)).toEqual({ min: 0.5, max: 2.5, step: 0.5 });
    expect(priceRange(18, 41)).toEqual({ min: 10, max: 50, step: 10 });
  });
});
