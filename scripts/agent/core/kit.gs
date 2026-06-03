import { createAgent } from "@agent/core/agent";
import { createRegistry } from "@agent/tools/registry";
import { createCodingTools } from "@agent/tools/coding";
import { createJSONLSession } from "@agent/session/jsonl";

export function createCodingAgent(options) {
  let registry = options.registry;
  if (!registry) {
    registry = createRegistry();
  }

  if (options.tools) {
    registry.registerAll(options.tools);
  }

  if (options.cwd && options.includeCodingTools !== false) {
    registry.registerAll(createCodingTools(options.cwd));
  }

  let session = options.session;
  if (!session && options.sessionFile) {
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
