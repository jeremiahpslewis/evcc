import { POWER_UNIT } from "../mixins/formatter";

export interface AxisScale {
  // below 1000 base units the axis shows the base unit (W, g) instead of the kilo unit
  small: boolean;
  digits: number;
  // nice-ceiled axis limit in base units
  limit: number;
}

export interface EnergyAxisScale {
  unit: POWER_UNIT.W | POWER_UNIT.KW;
  digits: number;
  // nice-ceiled axis limit in W(h)
  limit: number;
}

// Round up to a nice number (1/2/3/4/6/8 × 10^n).
export function niceCeil(v: number): number {
  if (v <= 0) return 0;
  const mag = Math.pow(10, Math.floor(Math.log10(v)));
  const r = v / mag;
  const n = r <= 1 ? 1 : r <= 2 ? 2 : r <= 3 ? 3 : r <= 4 ? 4 : r <= 6 ? 6 : r <= 8 ? 8 : 10;
  return n * mag;
}

// Shared y-axis scale for kilo-stepped units (kW, kWh, kg). Takes the data
// peak in base units: peaks below 1000 switch to the base unit with the axis
// floored at 1000 for stable context (zero data falls here too); kilo limits
// up to 3 get one decimal to avoid duplicate integer ticks.
export function axisScale(peak: number): AxisScale {
  const small = peak < 1000;
  const limit = small ? Math.max(niceCeil(peak), 1000) : niceCeil(peak);
  return { small, digits: !small && limit <= 3000 ? 1 : 0, limit };
}

export function energyAxisScale(peak: number): EnergyAxisScale {
  const { small, digits, limit } = axisScale(peak);
  return { unit: small ? POWER_UNIT.W : POWER_UNIT.KW, digits, limit };
}

export interface AxisRange {
  min: number;
  max: number;
  interval: number;
}

// Range for values above and below zero. Both sides step by half the limit of
// the larger side and end on the first step past their own peak, so a small
// export doesn't mirror the import range.
export function bidirectionalRange(upper: number, lower: number, limit: number): AxisRange {
  const interval = limit / 2;
  // tolerance keeps float noise on a step from adding another one
  const steps = (peak: number) => Math.max(0, Math.ceil(peak / interval - 1e-6));
  const up = steps(upper);
  const down = steps(lower);
  return {
    // an empty chart keeps its context above the baseline
    max: (up || down ? up : 2) * interval,
    // no -0, it would label the baseline "-0"
    min: down ? -down * interval : 0,
    interval,
  };
}
