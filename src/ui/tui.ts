import { Printer } from "./core.ts";
import { ExecutionResult } from "../types.ts";
import * as p from "@clack/prompts";
import colors from "picocolors";

export class TUI implements Printer {
  showIntro(count: number) {
    p.intro(colors.bgBlue(colors.white(` xn ${count} `)));
  }

  showOutro() {
    p.outro(colors.green("✓ Completed"));
  }

  printResult(result: ExecutionResult): void {
    const indent = "  ";

    // Exit code
    console.log(colors.dim("│"));
    console.log(
      colors.dim("│ ") + colors.bgMagenta(colors.white(" exit code ")) + " " +
        colors.bold(colors.white(result.exitCode.toString())),
    );

    // Stdout
    if (result.stdout) {
      console.log(colors.dim("│"));
      console.log(colors.dim("│ ") + colors.bgBlue(colors.white(" stdout ")));
      result.stdout.split("\n").forEach((line) => {
        console.log(colors.dim("│ ") + indent + colors.white(line));
      });
    }

    // Stderr
    if (result.stderr) {
      if (!result.stdout) {
        console.log(colors.dim("│"));
      }
      console.log(colors.dim("│ ") + colors.bgRed(colors.white(" stderr ")));
      result.stderr.split("\n").forEach((line) => {
        console.log(colors.dim("│ ") + indent + colors.red(line));
      });
    }

    if (!result.stdout && !result.stderr) {
      console.log(colors.dim("│"));
      console.log(colors.dim("│ ") + indent + colors.dim("(no output)"));
    }
  }
}
