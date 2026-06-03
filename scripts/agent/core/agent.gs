export function createAgent(options) {
  let provider = options.provider;
  let registry = options.registry;
  let session = options.session;
  let maxTurns = options.maxTurns;
  let onEvent = options.onEvent;

  if (maxTurns === undefined) {
    maxTurns = 8;
  }

  function emit(kind, payload) {
    let event = {
      kind: kind,
      payload: payload,
    };

    if (session !== undefined) {
      session.append(kind, payload);
    }

    if (onEvent !== undefined) {
      onEvent(event);
    }

    return event;
  }

  function run(input) {
    let messages = [
      { role: "user", content: input },
    ];

    emit("message", messages[0]);

    for (let turn = 0; turn < maxTurns; turn = turn + 1) {
      emit("turn_start", { turn: turn });
      let turnOptions = {};
      if (turn === maxTurns - 1) {
        turnOptions.toolChoice = "none";
      }
      let tools = registry.list();
      if (turnOptions.toolChoice === "none") {
        tools = [];
      }
      let next = provider.next(messages, tools, turnOptions);

      if (next.kind === "tool_call") {
        if (turnOptions.toolChoice === "none") {
          let forced = {
            role: "assistant",
            content: "Agent stopped before another tool call because maxTurns=" + String(maxTurns),
          };
          emit("message", forced);
          emit("turn_end", { turn: turn, stop: "tool_disabled" });
          return forced;
        }
        if (next.id === undefined) {
          next.id = "tool_" + String(turn);
        }
        emit("tool_call", next);
        messages.push(next);
        let result = registry.safeCall(next.name, next.args);
        let toolMessage = {
          role: "tool",
          id: next.id,
          name: next.name,
          content: JSON.stringify(result),
        };
        messages.push(toolMessage);
        emit("tool_result", toolMessage);
        emit("turn_end", { turn: turn, stop: "tool_call" });
        continue;
      }

      emit("message", next);
      emit("turn_end", { turn: turn, stop: "message" });
      return next;
    }

    let fallback = {
      role: "assistant",
      content: "Agent stopped after maxTurns=" + String(maxTurns),
    };
    emit("message", fallback);
    return fallback;
  }

  return {
    run: run,
  };
}
