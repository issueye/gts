let schema = require("@std/schema");
let crypto = require("@std/crypto");

let toolArgs = {
  type: "object",
  required: ["path"],
  additionalProperties: false,
  properties: {
    path: { type: "string", minLength: 1 },
    recursive: { type: "boolean" }
  }
};

let result = schema.validate(toolArgs, { path: "README.md", recursive: false });
let status = "invalid";
if (result.valid) {
  status = "valid";
}

println(status + ":" + crypto.sha256("agent").slice(0, 8));
