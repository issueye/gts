export function createFakeToolProvider(toolName, toolArgs, finalText) {
  return createScriptedProvider([
    {
      kind: "tool_call",
      name: toolName,
      args: toolArgs,
    },
    {
      role: "assistant",
      content: finalText,
    },
  ]);
}

export function createScriptedProvider(steps) {
  let step = 0;

  function next(messages, tools, turnOptions) {
    if (turnOptions !== undefined && turnOptions.toolChoice === "none") {
      return {
        role: "assistant",
        content: "No tool call requested.",
      };
    }

    if (step < steps.length) {
      let current = steps[step];
      step = step + 1;
      return current;
    }

    return {
      role: "assistant",
      content: "No scripted provider response remaining.",
    };
  }

  return {
    next: next,
  };
}
