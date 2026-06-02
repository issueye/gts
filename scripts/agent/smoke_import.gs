import { userMessage, assistantMessage } from "@agent/core/messages";

let user = userMessage("hello");
let assistant = assistantMessage("world");

println(user.role + ":" + user.content);
println(assistant.role + ":" + assistant.content);

