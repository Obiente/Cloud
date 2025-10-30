import { ExecutorContext, logger } from "@nx/devkit";
import { spawn } from "node:child_process";
import { isAbsolute, resolve } from "node:path";
import { existsSync } from "node:fs";
import { config } from "dotenv";
import type { RunShExecutorSchema } from "./schema";

interface SuccessResult {
  success: boolean;
}

const DEFAULT_COMMAND = "bash";

export default async function runShExecutor(
  options: RunShExecutorSchema,
  context: ExecutorContext
): Promise<SuccessResult> {
  const projectName = context.projectName ?? "unknown";
  const workspaceRoot = context.root ?? process.cwd();

  const projectRoot =
    context.projectsConfigurations?.projects?.[projectName]?.root ??
    context.projectGraph?.nodes?.[projectName]?.data?.root ??
    "";

  const cwdBase = options.cwd
    ? resolve(workspaceRoot, options.cwd)
    : projectRoot
    ? resolve(workspaceRoot, projectRoot)
    : workspaceRoot;

  // Load .env file if present
  const envFilePath = resolve(cwdBase, ".env");
  if (existsSync(envFilePath)) {
    config({ path: envFilePath });
    logger.info(`Loaded environment from ${envFilePath}`);
  }

  let resolvedScript: string | null = null;

  if (!options.inlineScript && options.script) {
    resolvedScript = isAbsolute(options.script)
      ? options.script
      : resolve(workspaceRoot, options.script);
  }

  if (
    !options.inlineScript &&
    (!resolvedScript || !existsSync(resolvedScript))
  ) {
    throw new Error(
      `run-sh executor: script not found at ${resolvedScript} (project: ${projectName}, cwd: ${cwdBase})`
    );
  }

  const useInterpreter = options.useInterpreter ?? true;
  const interpreter = options.command ?? DEFAULT_COMMAND;
  const scriptArgs = options.args ?? [];

  const command = options.inlineScript
    ? interpreter
    : useInterpreter
    ? interpreter
    : resolvedScript!;

  const args = options.inlineScript
    ? ["-c", options.inlineScript]
    : useInterpreter
    ? [resolvedScript!, ...scriptArgs]
    : [...scriptArgs];

  const env: NodeJS.ProcessEnv = {
    ...(options.forwardAllEnv ?? true ? process.env : {}),
    ...options.env,
  };

  logger.info(`▶ run-sh executor starting: ${command} ${args.join(" ")}`);
  logger.info(`📁 Working directory: ${cwdBase}`);

  return new Promise<SuccessResult>((resolvePromise, rejectPromise) => {
    const child = spawn(command, args, {
      cwd: cwdBase,
      env,
      stdio: "inherit",
      shell: options.shell ?? false,
    });

    const terminate = () => {
      if (!child.killed) {
        logger.warn(`⚠ Terminating script due to abort signal`);
        child.kill("SIGTERM");
      }
    };

    const abortSignal = (context as any)?.signal;
    abortSignal?.addEventListener?.("abort", terminate);

    child.on("error", (error) => {
      logger.error(`💥 run-sh executor failed: ${error.message}`);
      rejectPromise(error);
    });

    child.on("exit", (code, signal) => {
      if (signal) {
        logger.warn(`⚠ run-sh executor received signal: ${signal}`);
        resolvePromise({ success: false });
        return;
      }

      const success = code === 0;
      if (!success) {
        logger.error(`❌ run-sh executor exited with code ${code}`);
      } else {
        logger.info(`✅ run-sh executor completed successfully`);
      }

      resolvePromise({ success });
    });
  });
}
