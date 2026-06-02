export function createAgent(options) {
  let provider = options.provider;
  let registry = options.registry;
  let session = options.session;

  function run(input) {
    let messages = [
      { role: "user", content: input },
    ];

    if (session !== undefined) {
      session.append("message", messages[0]);
    }

    let first = provider.next(messages, registry.list());
    if (first.kind === "tool_call") {
      if (session !== undefined) {
        session.append("tool_call", first);
      }

      let result = registry.call(first.name, first.args);
      let toolMessage = {
        role: "tool",
        name: first.name,
        content: JSON.stringify(result),
      };
      messages.push(toolMessage);

      if (session !== undefined) {
        session.append("tool_result", toolMessage);
      }

      let finalMessage = provider.next(messages, registry.list());
      if (session !== undefined) {
        session.append("message", finalMessage);
      }
      return finalMessage;
    }

    if (session !== undefined) {
      session.append("message", first);
    }
    return first;
  }

  return {
    run: run,
  };
}
