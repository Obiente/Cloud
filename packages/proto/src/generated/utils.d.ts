import { type Timestamp } from "@bufbuild/protobuf/wkt";
/**
 * Create a protobuf `Timestamp` from a JavaScript `Date`.
 * Produces `{ seconds: bigint, nanos: number }` matching the generated types.
 */
export declare function timestamp(date: Date): Timestamp;
/**
 * Convert a protobuf `Timestamp` to a JavaScript `Date`.
 * Assumes `Timestamp.seconds` is a `bigint` and `nanos` is a number.
 */
export declare function date(ts: Timestamp): Date;
