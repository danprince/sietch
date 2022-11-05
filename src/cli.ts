#!/usr/bin/env node

import { build, siteFromDir } from "./builder";
import { serve } from "./server";
import { bold } from "picocolors";
import { time } from "./helpers";
import { reportError } from "./errors";

export async function cli() {
  let cmd = process.argv[2] || "build";
  console.log(bold("sietch"), cmd);

  let site = await siteFromDir(process.cwd());

  if (cmd === "dev") {
    site.env = "development";
    await serve(site);
  } else {
    site.env = "production";
    try {
      await time("built in", build(site));
    } catch (err: any) {
      reportError(err);
    }
  }
}

cli();
