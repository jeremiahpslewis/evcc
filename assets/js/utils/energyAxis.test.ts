import { describe, expect, test } from "vite-plus/test";
import { POWER_UNIT } from "../mixins/formatter";
import { axisScale, bidirectionalRange, energyAxisScale, niceCeil } from "./energyAxis";

describe("niceCeil", () => {
  test("rounds up to nice numbers", () => {
    expect(niceCeil(0)).toBe(0);
    expect(niceCeil(60)).toBe(60);
    expect(niceCeil(5)).toBe(6);
    expect(niceCeil(700)).toBe(800);
    expect(niceCeil(1000)).toBe(1000);
    expect(niceCeil(15500)).toBe(20000);
  });
});

describe("axisScale", () => {
  test("steps between base and kilo unit at 1000", () => {
    expect(axisScale(402)).toEqual({ small: true, digits: 0, limit: 1000 });
    expect(axisScale(2500)).toEqual({ small: false, digits: 1, limit: 3000 });
  });
});

describe("energyAxisScale", () => {
  test("switches to W below 1 kW and floors the axis at 1000", () => {
    expect(energyAxisScale(600)).toEqual({ unit: POWER_UNIT.W, digits: 0, limit: 1000 });
  });

  test("zero data uses W scale", () => {
    expect(energyAxisScale(0)).toEqual({ unit: POWER_UNIT.W, digits: 0, limit: 1000 });
  });

  test("1-3 kW band gets one decimal to avoid duplicate integer ticks", () => {
    expect(energyAxisScale(1000)).toEqual({ unit: POWER_UNIT.KW, digits: 1, limit: 1000 });
    expect(energyAxisScale(2500)).toEqual({ unit: POWER_UNIT.KW, digits: 1, limit: 3000 });
  });

  test("larger peaks use kW without decimals", () => {
    expect(energyAxisScale(3500)).toEqual({ unit: POWER_UNIT.KW, digits: 0, limit: 4000 });
    expect(energyAxisScale(15500)).toEqual({ unit: POWER_UNIT.KW, digits: 0, limit: 20000 });
  });
});

describe("bidirectionalRange", () => {
  test("smaller side ends on the first step past its peak", () => {
    expect(bidirectionalRange(57, 22, 60)).toEqual({ min: -30, max: 60, interval: 30 });
    expect(bidirectionalRange(1.4, 1.0, 2)).toEqual({ min: -1, max: 2, interval: 1 });
  });

  test("sides close to each other stay symmetric", () => {
    expect(bidirectionalRange(2.4, 2.0, 3)).toEqual({ min: -3, max: 3, interval: 1.5 });
  });

  test("float noise on a step adds no further step", () => {
    expect(bidirectionalRange(2, 1.0000000001, 2)).toEqual({ min: -1, max: 2, interval: 1 });
  });

  test("one direction only", () => {
    expect(bidirectionalRange(57, 0, 60)).toEqual({ min: 0, max: 60, interval: 30 });
    expect(bidirectionalRange(0, 1.2, 2)).toEqual({ min: -2, max: 0, interval: 1 });
  });

  test("empty chart keeps the range above the baseline", () => {
    expect(bidirectionalRange(0, 0, 1)).toEqual({ min: 0, max: 1, interval: 0.5 });
  });
});
