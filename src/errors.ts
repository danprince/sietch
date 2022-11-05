import { bold, red } from "picocolors";

let errorTitle = bold(red("error"));

export function reportError(error: any) {
  if (error.errors && error.warnings) {
    console.error(errorTitle, "esbuild error");
  } else if (error instanceof Error) {
    console.error(errorTitle, error.stack || error.message);
  }
}
