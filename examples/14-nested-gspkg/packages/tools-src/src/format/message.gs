import { bracket, suffix } from "helper";

export function formatMessage(topic) {
  return bracket("tools -> " + topic + " -> " + suffix);
}
