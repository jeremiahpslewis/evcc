// how close a prediction came, from per bucket [predicted, actual] pairs (null where no
// prediction was recorded): the absolute error against the predicted total, so misses in
// both directions do not cancel out, and how far the predicted total sat above the actual
// one, negative below. Null without a prediction.
export function predictionAccuracy(
  spans: ([number, number] | null)[]
): { accuracy: number; delta: number } | null {
  const known = spans.filter((s): s is [number, number] => s !== null);
  const predicted = known.reduce((acc, s) => acc + s[0], 0);
  if (predicted <= 0) return null;
  const error = known.reduce((acc, s) => acc + Math.abs(s[0] - s[1]), 0);
  const actual = known.reduce((acc, s) => acc + s[1], 0);
  return { accuracy: 1 - error / predicted, delta: predicted - actual };
}

// per bucket [predicted, actual], null where either was not recorded: a bucket the
// actual did not cover (a gap in recording) says nothing about the prediction
export function pairBuckets(
  predicted: number[],
  actual: number[],
  hasPrediction: Set<number>,
  hasActual: Set<number>
): ([number, number] | null)[] {
  return predicted.map((p, i): [number, number] | null =>
    hasPrediction.has(i) && hasActual.has(i) ? [p, actual[i] || 0] : null
  );
}

// how close a prediction came where metered parts of it, such as a heat pump, run when they
// are planned rather than when predicted: the rest of the prediction is scored per bucket, each
// part by its energy per scoring group of buckets (a day). total holds per bucket the predicted
// and actual consumption, null where either was not recorded; parts the same per part, a bucket
// without a part's pair keeps that part in the rest. Without parts it scores as predictionAccuracy.
export function splitAccuracy(
  total: ([number, number] | null)[],
  parts: ([number, number] | null)[][],
  group: (bucket: number) => number
): { accuracy: number; delta: number } | null {
  let predicted = 0;
  let actual = 0;
  let error = 0;
  const partSums = new Map<string, [number, number]>();

  total.forEach((pair, i) => {
    if (!pair) return;
    let [restPredicted, restActual] = pair;
    predicted += pair[0];
    actual += pair[1];

    parts.forEach((part, p) => {
      const partPair = part[i];
      if (!partPair) return;
      restPredicted -= partPair[0];
      restActual -= partPair[1];
      const key = `${p}:${group(i)}`;
      const sum = partSums.get(key) || [0, 0];
      partSums.set(key, [sum[0] + partPair[0], sum[1] + partPair[1]]);
    });

    error += Math.abs(restPredicted - restActual);
  });

  if (predicted <= 0) return null;
  partSums.forEach(([p, a]) => (error += Math.abs(p - a)));
  return { accuracy: 1 - error / predicted, delta: predicted - actual };
}
