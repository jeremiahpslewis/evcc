// how close a prediction came, from per bucket [predicted, actual] pairs (null where no
// prediction was recorded): the absolute error against the actual total, so misses in
// both directions do not cancel out and a prediction that runs low is not marked down
// for it beyond its error (against the predicted total, one that ran high would score
// better for the same error), and how far the predicted total sat above the actual one,
// negative below. Null without a prediction or without anything consumed.
export function predictionAccuracy(
  spans: ([number, number] | null)[]
): { accuracy: number; delta: number } | null {
  const known = spans.filter((s): s is [number, number] => s !== null);
  const predicted = known.reduce((acc, s) => acc + s[0], 0);
  const actual = known.reduce((acc, s) => acc + s[1], 0);
  if (predicted <= 0 || actual <= 0) return null;
  const error = known.reduce((acc, s) => acc + Math.abs(s[0] - s[1]), 0);
  return { accuracy: 1 - error / actual, delta: predicted - actual };
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
