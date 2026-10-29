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

A load forecast (`tariffs.load`, a custom tariff with `tariff: load`) replaces the profile for the slots it covers. Its value is the mean household power in W per rate without the loadpoints, which are modelled separately; rates may be of any length. Slots beyond the forecast keep the profile. While the forecast is active the published result's `details` carry `loadForecast`, one boolean per slot aligned with `details.timestamp` and true where the forecast replaced the profile (a slot merged to an hour beyond 48h is true if any of its quarters was), and `homeProfile`, the replaced profile in Wh per slot for comparison. Both are omitted when the solve used the profile throughout: no forecast configured, the forecast stale after twice its interval, or none of its rates within the horizon. The forecast and the profile are taken as they stand in every slot, the first included: the first slot, from the solve to the next quarter hour, takes the running quarter hour's power over its remaining seconds.

Every solve reads a custom load forecast anew before it plans; a failed read keeps the rates read last while they are current. A custom forecast may name `generatedAt`, a plugin reading when the forecast was issued (e.g. the attribute `generated_at` of the same entity). It is read before and after the forecast, which is read again when it changed in between, so that rates and issue belong together. Every 30 seconds it is read on its own, and an issue the last solve did not read starts a solve, so that the plan follows a new issue within a minute. The published `details` then carry `loadForecastGeneratedAt`, the `generatedAt` of the issue planned with exactly as read, beside `loadForecast`, and leave it out with it.

Both predictions are recorded per slot, the forecast whenever one is configured, the profile while the optimizer is enabled or a forecast replaces it (metrics groups `homeforecast` and `homeprofile`). The Energy view compares them to the consumption.

A custom load forecast may name `parts`, forecasts of loads it contains that have an aux or consumer meter of their own, keyed by the meter's name, such as a heat pump whose runs are planned after the forecast is made. The optimizer still uses the forecast itself. Each part's forecast and its meter's default profile are recorded per slot (`consumerforecast` and `consumerprofile`, titled like the meter), and the Energy view scores the rest of a prediction per bucket and each part by its energy per day (per month in the year view) against its meter, so the parts' timing does not count against the forecast.

### Measured value blending

The solar forecast is anchored to the current situation using the last completed 15min
metrics slot, decaying linearly over 4 slots: the scale factor measured production/forecasted
production is applied to the first slot and decays towards 1. The base load is not blended
with the measured home consumption.

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
