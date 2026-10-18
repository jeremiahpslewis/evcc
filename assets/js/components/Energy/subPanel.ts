import colors, { setAlpha } from "@/colors";
import { forecastYAxis, hoverDot, lineDefaults } from "./echarts";

// skinny panel below a chart's main plot, sharing its x axis
const PANEL_HEIGHT = 40;
const PANEL_GAP = 24;
export const PANEL_EXTRA = 48; // added chart height, the main plot gives up the rest

export interface SubPanel {
  series: Record<string, unknown>[];
  yAxis: Record<string, unknown>;
  track: string; // background of the panel, the full range
}

// Price range widened to whole steps of 1, 2 or 5 × 10^n, about a quarter of the
// span each, so both ends read as round numbers in any currency
export function priceRange(lo: number, hi: number): { min: number; max: number; step: number } {
  const rough = (hi - lo) / 4;
  const mag = Math.pow(10, Math.floor(Math.log10(rough)));
  // a span like 0.32 - 0.3 carries float noise that would skip a step size
  const r = Number((rough / mag).toPrecision(12));
  const step = (r <= 1 ? 1 : r <= 2 ? 2 : r <= 5 ? 5 : 10) * mag;
  // tolerance keeps float noise on a step from adding another one, precision drops
  // the noise of the multiplication
  const at = (n: number) => Number((n * step).toPrecision(12));
  return {
    min: at(Math.floor(lo / step + 1e-6)),
    max: at(Math.ceil(hi / step - 1e-6)),
    step: Number(step.toPrecision(12)),
  };
}

// soc or temperature per category as line and area on grid, x and y axis 1
export function socTempPanel(
  values: (number | null)[],
  color: string,
  isTemp: boolean,
  format: (v: number) => string
): SubPanel {
  // soc spans 0 to 100, temperature the data's range widened to full tens
  let [min, max] = [0, 100];
  if (isTemp) {
    const present = values.filter((v): v is number => v !== null);
    min = Math.floor(Math.min(...present) / 10) * 10;
    max = Math.max(Math.ceil(Math.max(...present) / 10) * 10, min + 10);
  }
  // a light box needs less tint than a dark one
  const dark = document.documentElement.classList.contains("dark");
  return {
    // same hue as the area, so the fill stays a clean tint of the device color
    track: setAlpha(color, dark ? "1a" : "0d") || "",
    series: [
      {
        id: "soctemp",
        name: "soctemp",
        type: "line",
        xAxisIndex: 1,
        yAxisIndex: 1,
        data: values,
        smooth: true,
        ...hoverDot(color),
        connectNulls: true,
        lineStyle: { color, ...lineDefaults },
        areaStyle: { color: setAlpha(color, "40") },
        z: 4,
      },
    ],
    yAxis: forecastYAxis({
      gridIndex: 1,
      position: "left",
      min,
      max,
      // labels at both ends only
      interval: max - min,
      // the grid's background is the track, a second set of lines would fight the main grid
      splitLine: { show: false },
      axisLabel: { color: colors.muted || "", hideOverlap: true, formatter: format },
    }),
  };
}

// the main plot's grid shortened, and the panel's grid below it
export function panelGrids(main: { bottom: number; left: number; right: number }, track: string) {
  return [
    { ...main, bottom: main.bottom + PANEL_HEIGHT + PANEL_GAP },
    {
      bottom: main.bottom,
      height: PANEL_HEIGHT,
      left: main.left,
      right: main.right,
      borderWidth: 0,
      show: true,
      backgroundColor: track,
    },
  ];
}
