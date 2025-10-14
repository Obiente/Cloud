export interface RunShExecutorSchema {
  script: string;
  args?: string[];
  command?: string;
  useInterpreter?: boolean;
  cwd?: string;
  env?: Record<string, string>;
  forwardAllEnv?: boolean;
}
