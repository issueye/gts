import { createTool } from "@agent/tools/registry";
import { workspacePath } from "@agent/tools/files";

let exec = require("@std/exec");
let os = require("@std/os");

function shellCommand(command) {
  if (os.platform === "windows") {
    return exec.command("powershell", ["-NoProfile", "-Command", command]);
  }
  return exec.command("bash", ["-lc", command]);
}

export function createBashTool(cwd) {
  return createTool(
    "bash",
    "Run a shell command in the workspace and return stdout, stderr, and exit code.",
    {
      type: "object",
      required: ["command"],
      additionalProperties: false,
      properties: {
        command: { type: "string", minLength: 1 },
        cwd: { type: "string" },
      },
    },
    function(args) {
      let runDir = cwd;
      if (args.cwd !== undefined) {
        runDir = workspacePath(cwd, args.cwd);
      }
      let cmd = shellCommand(args.command);
      cmd.setDir(runDir);
      return cmd.run();
    }
  );
}
