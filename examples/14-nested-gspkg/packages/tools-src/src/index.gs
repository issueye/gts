import { bracket, suffix } from "helper";

export function buildMessage(topic) {
  return bracket("tools -> " + topic + " -> " + suffix);
}
