// Local minimal type shim for `@opencode-ai/plugin`, used ONLY for the repo typecheck. The real
// package (verified against v1.17.15 installed under ~/.config/opencode) is provided by the
// OpenCode runtime; this adapter imports it TYPE-ONLY, so nothing here ships or runs. Keep this in
// sync with the subset of the real `Hooks`/`Config` surface this adapter uses.

declare module "@opencode-ai/plugin" {
  /** The subset of an MCP entry this adapter writes. */
  export type McpLocalConfig = {
    type: "local";
    command: string[];
    enabled?: boolean;
    environment?: Record<string, string>;
    timeout?: number;
  };
  export type McpRemoteConfig = {
    type: "remote";
    url: string;
    enabled?: boolean;
  };

  /** The OpenCode config object passed to the `config` hook (only the fields this adapter reads). */
  export type Config = {
    mcp?: Record<string, McpLocalConfig | McpRemoteConfig>;
    [key: string]: unknown;
  };

  export interface Hooks {
    config?: (input: Config) => Promise<void>;
    dispose?: () => Promise<void>;
    event?: (input: { event: unknown }) => Promise<void>;
    [key: string]: unknown;
  }

  export type PluginInput = {
    directory: string;
    worktree: string;
    [key: string]: unknown;
  };
  export type PluginOptions = Record<string, unknown>;

  export type Plugin = (input: PluginInput, options?: PluginOptions) => Promise<Hooks>;
}
