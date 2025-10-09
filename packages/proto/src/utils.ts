import { create } from "@bufbuild/protobuf";
import { Timestamp, TimestampSchema } from "@bufbuild/protobuf/wkt";

export function timestamp(date: Date): Timestamp {
    const seconds = BigInt(Math.floor(date.getTime() / 1000));
    const nanos = date.getMilliseconds() * 1_000_000;
    return create(TimestampSchema, { seconds, nanos });
}

