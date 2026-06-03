import { createAgent } from "@agent/core/agent";
import { createRegistry } from "@agent/tools/registry";
import { createCodingTools } from "@agent/tools/coding";
import { createJSONLSession } from "@agent/session/jsonl";

export function createCodingAgent(options) {
  let registry = options.registry;
  if (registry === undefined) {
    registry = createRegistry();
  }

  if (options.tools !== undefined) {
    registry.registerAll(options.tools);
  }

  if (options.cwd !== undefined && options.includeCodingTools !== false) {
    registry.registerAll(createCodingTools(options.cwd));
  }

  let session = options.session;
  if (session === undefined && options.sessionFile !== undefined) {
    session = createJSONLSession(options.sessionFile);
  }

  let agent = createAgent({
    provider: options.provider,
    registry: registry,
    session: session,
    maxTurns: options.maxTurns,
    onEvent: options.onEvent,
  });

  return {
    agent: agent,
    registry: registry,
    session: session,
  };
}
