// @ts-nocheck
import { moo } from "https://esm.sh/cowsayjs@1.0.7";

export function render({ message }: { message: string }): string {
  return moo(message);
}

