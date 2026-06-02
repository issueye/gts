let toml = require("@std/toml");
let yaml = require("@std/yaml");
let xml = require("@std/xml");

let t = toml.parse("[agent]\nname = \"coder\"\ntools = [\"read\", \"write\"]\n");
let y = yaml.parse("agent:\n  name: coder\n  enabled: true\n");
let x = xml.parse("<agent name=\"coder\"><tool>read</tool></agent>");

let xmlText = xml.stringify(x);
let kind = "bad";
if (t.agent.tools.length === 2 && y.agent.enabled && x.children[0].text === "read" && xmlText.includes("<agent")) {
  kind = "ok";
}

println("config-codecs:" + kind);
