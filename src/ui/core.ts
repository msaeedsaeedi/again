import { ExecutionResult } from "../types.ts";
import colors from "picocolors";

export interface Printer {
  showIntro(count: number): void;
  showOutro(): void;
  printResult(result: ExecutionResult): void;
}
export function showVersion(version: string) {
  console.log(colors.green(`xn version ${version}`));
}

export function showHelp() {
  console.log(
    colors.bold(colors.cyan("Usage:")) + " " + colors.white("xn [options] <command>") + "\n" +
      "\n" +
      colors.bold(colors.cyan("Options:")) + "\n" +
      "  " + colors.yellow("-n") + ", " + colors.yellow("--count") + " <number>   " +
      colors.dim("Number of times to execute the command (default: 1)") + "\n" +
      "  " + colors.yellow("-s") + ", " + colors.yellow("--silent") + "           " +
      colors.dim("Silent mode - suppress output") + "\n" +
      "  " + colors.yellow("-h") + ", " + colors.yellow("--help") + "             " +
      colors.dim("Show help information") + "\n" +
      "  " + colors.yellow("-v") + ", " + colors.yellow("--version") + "          " +
      colors.dim("Show version information") + "\n" +
      "\n" +
      colors.bold(colors.cyan("Example:")) + "\n" +
      "  " + colors.green('xn -n 5 "echo Hello, World!"'),
  );
}
