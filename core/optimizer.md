# Optimizer

Optimizer uses mixed integer linear programming (MILP) to minize a cost function. The optimizer model implementation honors:

- energy consumption/ feed-in costs
- solar forecast
- base load (aka home) energy demand
- end of forecast commercial value
- strategy- either "charge before export" (charge loads as soon as possible) or attenuating grid peaks on the demand side, the feed-in side or both
- home battery or loadpoint/vehicle...
  - capacity, soc and charge goals
  - charge/discharge power limits and efficiency

Optimization spans N slots with N being minimum of available forecast data.

## Parameters

### Energy consumption/ feed-in costs/ Solar forecast

Grid/ feed-in/ solar tariff data.

TODO

- [ ] make feed-in optional
- [ ] make solar optional

### Base load energy demand

Collected 15min energy profile over the last 4 weeks, averaged per slot. A per-slot percentile (e.g. 50 = median, robust against a few heavy days) can be configured via `/api/profilepercentile` instead.

A load forecast (`tariffs.load`, a custom tariff with `tariff: load`) replaces the profile for the slots it covers. Its value is the mean household power in W per rate without the loadpoints, which are modelled separately; rates may be of any length. Slots beyond the forecast keep the profile. While the forecast is active the published result's `details` carry `loadForecast`, one boolean per slot aligned with `details.timestamp` and true where the forecast replaced the profile (a slot merged to an hour beyond 48h is true if any of its quarters was), and `homeProfile`, the replaced profile in Wh per slot for comparison. Both are omitted when the solve used the profile throughout: no forecast configured, the forecast stale after twice its interval, or none of its rates within the horizon. The measured value blending below applies to either.

Both predictions are recorded per slot, the forecast whenever one is configured, the profile while the optimizer is enabled or a forecast replaces it (metrics groups `homeforecast` and `homeprofile`). The Energy view compares them to the consumption.

A custom load forecast may name `parts`, forecasts of loads it contains that have an aux or consumer meter of their own, keyed by the meter's name, such as a heat pump whose runs are planned after the forecast is made. The optimizer still uses the forecast itself. Each part's forecast and its meter's default profile are recorded per slot (`consumerforecast` and `consumerprofile`, titled like the meter), and the Energy view scores the rest of a prediction per bucket and each part by its energy per day (per month in the year view) against its meter, so the parts' timing does not count against the forecast.

### Measured value blending

The solar forecast and the base load profile are anchored to the current situation
using the last completed 15min metrics slot, decaying linearly over 4 slots:

- base load: the measured home consumption replaces the first slot and decays into the profile
- solar: the scale factor measured production/forecasted production is applied to the first slot and decays towards 1

### End of forecast commercial value

Use minimum of energy consumption cost.

### Home Battery

### Loadpoint and Vehicles

- home battery or loadpoint/vehicle...
  - capacity, soc and charge goals
  - charge/discharge power limits and efficiency

Without vehicle capacity or soc a configured session energy limit is modelled instead:
state is the session's charged energy, goal is the limit. Loadpoints with neither are
not modelled at all- their power is added to the base load.
