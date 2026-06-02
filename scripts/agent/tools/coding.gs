import { createFileTools } from "@agent/tools/files";
import { createBashTool } from "@agent/tools/bash";
import { createGrepTool } from "@agent/tools/grep";

export function createCodingTools(cwd) {
  let tools = createFileTools(cwd);
  tools.push(createBashTool(cwd));
  tools.push(createGrepTool(cwd));
  return tools;
}
