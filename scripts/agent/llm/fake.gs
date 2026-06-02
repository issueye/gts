export function createFakeToolProvider(toolName, toolArgs, finalText) {
  let step = 0;

  function next(messages, tools) {
    if (step === 0) {
      step = 1;
      return {
        kind: "tool_call",
        name: toolName,
        args: toolArgs,
      };
    }

    return {
      role: "assistant",
      content: finalText,
    };
  }

  return {
    next: next,
  };
}
