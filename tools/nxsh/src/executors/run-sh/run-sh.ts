import { ExecutorContext, logger } from "@nx/devkit";
import { isAbsolute, resolve } from "node:path";
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import type { RunShExecutorSchema } from "./schema";

interface SuccessResult {
  success: boolean;
}

const DEFAULT_COMMAND = "bash";

export default async function runShExecutor(
  options: RunShExecutorSchema,
  context: ExecutorContext
): Promise<SuccessResult> {
  const projectName = context.projectName;
  const workspaceRoot = context.root ?? process.cwd();
  const projectRoot = projectName
    ? context.projectsConfigurations?.projects?.[projectName]?.root ??
      context.projectGraph?.nodes?.[projectName]?.data?.root ??
      ""
    : "";

  const cwdBase = options.cwd
    ? resolve(workspaceRoot, options.cwd)
    : projectRoot
    ? resolve(workspaceRoot, projectRoot)
    : workspaceRoot;

  const resolvedScript = isAbsolute(options.script)
    ? options.script
    : resolve(workspaceRoot, options.script);

  if (!existsSync(resolvedScript)) {
    throw new Error(`run-sh executor: script not found at ${resolvedScript}`);
  }

  const useInterpreter = options.useInterpreter ?? true;
  const interpreter = options.command ?? DEFAULT_COMMAND;
  const scriptArgs = options.args ?? [];

  const command = useInterpreter ? interpreter : resolvedScript;
  const args = useInterpreter
    ? [resolvedScript, ...scriptArgs]
    : [...scriptArgs];

  const env = {
    ...(options.forwardAllEnv ?? true ? process.env : {}),
    ...options.env,
  } as NodeJS.ProcessEnv;

  logger.info(`run-sh executor starting: ${command} ${args.join(" ")}`.trim());
  logger.info(`working directory: ${cwdBase}`);

  return new Promise<SuccessResult>((resolvePromise, rejectPromise) => {
    const child = spawn(command, args, {
      cwd: cwdBase,
      env,
      stdio: "inherit",
    });

    const terminate = () => {
      if (!child.killed) {
        child.kill("SIGTERM");
      }
    };

    const abortSignal = (context as any)?.signal as any;
    abortSignal?.addEventListener?.("abort", terminate);

    child.on("error", (error) => {
      logger.error(`run-sh executor failed: ${error.message}`);
      rejectPromise(error);
    });

    child.on("exit", (code, signal) => {
      if (signal) {
        logger.warn(`run-sh executor received signal ${signal}`);
        resolvePromise({ success: false });
        return;
      }

      const success = code === 0;
      if (!success) {
        logger.error(`run-sh executor exited with code ${code}`);
      }
      resolvePromise({ success });
    });
  });
}
