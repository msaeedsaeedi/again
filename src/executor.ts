import type { CliOptions, ExecutionResult } from "./types.ts";
import * as clack from "@clack/prompts";
import { Printer } from "./ui/core.ts";
import { TUI } from "./ui/tui.ts";

/**
 * Get the appropriate shell command based on the OS
 */
function getShellCommand(command: string): string[] {
  if (Deno.build.os === "windows") {
    return ["cmd.exe", "/c", command];
  } else {
    return ["/bin/sh", "-c", command];
  }
}

/**
 * Execute a command and capture its output
 */
async function executeCommand(
  command: string,
  index: number,
): Promise<ExecutionResult> {
  const startTime = performance.now();

  try {
    const shellCommand = getShellCommand(command);
    const cmd = new Deno.Command(shellCommand[0], {
      args: shellCommand.slice(1),
      stdout: "piped",
      stderr: "piped",
    });

    const process = cmd.spawn();
    const { code, stdout, stderr } = await process.output();

    const duration = performance.now() - startTime;

    return {
      index,
      exitCode: code,
      stdout: new TextDecoder().decode(stdout),
      stderr: new TextDecoder().decode(stderr),
      duration,
      success: code === 0,
    };
  } catch (error) {
    const duration = performance.now() - startTime;
    return {
      index,
      exitCode: -1,
      stdout: "",
      stderr: error instanceof Error ? error.message : String(error),
      duration,
      success: false,
    };
  }
}

/**
 * Main execution function
 */
export async function execute(options: CliOptions): Promise<void> {
  const printer: Printer = new TUI();
  let spinner: ReturnType<typeof clack.spinner> | null = null;

  printer.showIntro(options.count);

  Deno.addSignalListener("SIGTSTP", () => {
    if (spinner) spinner.stop("Canceled");
    Deno.exit(0);
  });

  for (let i = 1; i <= options.count; i++) {
    spinner = clack.spinner();

    spinner.start("Executing command...");
    const result = await executeCommand(options.command, i);
    spinner.stop("Execution completed.", result.exitCode);

    if (!options.silent) {
      printer.printResult(result);
    }
  }
  printer.showOutro();
}
