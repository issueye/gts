export function userMessage(text) {
  return { role: "user", content: text };
}

export function assistantMessage(text) {
  return { role: "assistant", content: text };
}

