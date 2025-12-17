/**
 * CLI configuration options
 */
export interface CliOptions {
  /** Number of times to execute the command */
  count: number;
  /** Command or script to execute */
  command: string;
  /** Silent mode - suppress output */
  silent: boolean;
}

/**
 * Result of a single command execution
 */
export interface ExecutionResult {
  /** Execution index (1-based) */
  index: number;
  /** Exit code returned by the command */
  exitCode: number;
  /** Standard output */
  stdout: string;
  /** Standard error */
  stderr: string;
  /** Execution duration in milliseconds */
  duration: number;
  /** Whether the execution was successful (exit code 0) */
  success: boolean;
}
