import { create } from "@bufbuild/protobuf";
import { type Timestamp, TimestampSchema } from "@bufbuild/protobuf/wkt";

/**
 * Create a protobuf `Timestamp` from a JavaScript `Date`.
 * Produces `{ seconds: bigint, nanos: number }` matching the generated types.
 */
export function timestamp(date: Date): Timestamp {
  const seconds = BigInt(Math.floor(date.getTime() / 1000));
  const nanos = date.getMilliseconds() * 1_000_000;
  return create(TimestampSchema, { seconds, nanos });
}

/**
 * Convert a protobuf `Timestamp` to a JavaScript `Date`.
 * Assumes `Timestamp.seconds` is a `bigint` and `nanos` is a number.
 */
export function date(ts: Timestamp): Date {
  if (!ts) return new Date(0);

  const secondsBig: bigint = ts.seconds;
  const nanosNum: number = ts.nanos;

  // milliseconds = seconds * 1000 + floor(nanos / 1e6)
  const msBig = secondsBig * 1000n + BigInt(Math.floor(nanosNum / 1_000_000));

  return new Date(Number(msBig));
}
