import { parseArgs } from "jsr:@std/cli@1/parse-args";
import { CliOptions } from "./types.ts";
import { execute } from "./executor.ts";
import { showHelp, showVersion } from "./ui/core.ts";
import { config } from "./config.ts";

/**
 * Parse CLI arguments
 */
function parseCliArgs(): CliOptions | null {
  const args = parseArgs(Deno.args, {
    string: ["count"],
    boolean: ["silent", "help", "version"],
    alias: {
      s: "silent",
      h: "help",
      v: "version",
      n: "count",
    },
  });

  if (args.help) {
    showHelp();
    return null;
  }

  if (args.version) {
    showVersion(config.version);
    return null;
  }

  const count = parseInt(String(args.count || args.n || "1"));
  const command = args._[0] as string;

  if (!command) {
    showHelp();
    Deno.exit(1);
  }

  return {
    count,
    command,
    silent: Boolean(args.silent || args.s),
  };
}

/**
 * Main execution
 */
async function main(): Promise<void> {
  const options: CliOptions | null = parseCliArgs();

  if (!options) {
    return;
  }

  await execute(options);
  Deno.exit(0);
}

if (import.meta.main) {
  main().catch((error) => {
    console.error("An unexpected error occurred:", error);
    Deno.exit(1);
  });
}
