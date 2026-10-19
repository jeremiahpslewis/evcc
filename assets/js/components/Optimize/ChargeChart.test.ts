import { mount, config } from "@vue/test-utils";
import { defineComponent } from "vue";
import { describe, expect, test } from "vite-plus/test";
import ChargeChart from "./ChargeChart.vue";
import type { EvoptData } from "./TimeSeriesDataTable.vue";
import {
  batteryDetails,
  evopt,
  home,
  homeForecast,
  homeProfile,
  loadForecast,
  timestamps,
} from "./stories.fixture";

config.global.mocks["$i18n"] = { locale: "en" };
config.global.mocks["$t"] = (a: string) => a;

// echarts needs a canvas, the computed chart options are what matter here
const TestChart = defineComponent({ extends: ChargeChart, methods: { initChart() {} } });

function chart(data: EvoptData, profile?: number[], covered?: boolean[]) {
  return mount(TestChart, {
    props: {
      evopt: data,
      batteryDetails,
      timestamp: timestamps[0],
      homeProfile: profile,
      loadForecast: covered,
    },
  }).vm as unknown as {
    legends: { label: string; type?: string }[];
    stackEntries: { label: string; data: number[] }[];
    chartSeries: {
      name?: string;
      type?: string;
      data?: (number | null)[];
      areaStyle?: { opacity?: number };
    }[];
  };
}

const FORECAST = "Consumption (forecast)";
const DEFAULT = "Consumption (default)";
const CONSUMPTION = "energy.group.consumer";

describe("consumption forecast", () => {
  test("learned profile only: plain consumption, no comparison line", () => {
    const vm = chart(evopt([home]));
    expect(vm.legends.map((l) => l.label)).not.toContain(DEFAULT);
    expect(vm.stackEntries[0]!.label).toBe(CONSUMPTION);
  });

  test("forecast slots and profile slots are separate bars, never negative", () => {
    const vm = chart(evopt([homeForecast]), homeProfile, loadForecast);
    const forecast = vm.stackEntries.find((e) => e.label === FORECAST)!;
    const rest = vm.stackEntries.find((e) => e.label === DEFAULT)!;

    loadForecast.forEach((covered, i) => {
      expect(covered ? rest.data[i] : forecast.data[i]).toBe(0);
    });
    expect(Math.min(...forecast.data, ...rest.data)).toBeGreaterThanOrEqual(0);
  });

  test("default is greyed out over the bars where the forecast replaced it only", () => {
    const vm = chart(evopt([homeForecast]), homeProfile, loadForecast);
    const labels = vm.legends.map((l) => l.label);
    expect(labels).toEqual([...new Set(labels)]);
    expect(labels).toContain(FORECAST);
    expect(labels).toContain(DEFAULT);

    const ghost = vm.chartSeries.find((s) => s.name === DEFAULT && s.type === "line")!;
    expect(ghost.areaStyle?.opacity).toBeLessThan(1);
    // quarter hour energy (Wh) to power (kW)
    expect(ghost.data?.[0]).toBeCloseTo(homeProfile[0]! / 0.25 / 1000);
    expect(ghost.data?.[loadForecast.length - 1]).toBeNull();
  });

  test("forecast over the whole horizon lists the dashed default itself", () => {
    const vm = chart(
      evopt([homeForecast]),
      homeProfile,
      loadForecast.map(() => true)
    );
    expect(vm.stackEntries.map((e) => e.label)).not.toContain(DEFAULT);
    expect(vm.legends).toContainEqual(expect.objectContaining({ label: DEFAULT, type: "line" }));
  });
});
