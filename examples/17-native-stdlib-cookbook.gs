// ============================================================
// 17-native-stdlib-cookbook.gs -- 原生标准库 Cookbook
// ============================================================
// 这些模块由 Go 侧注册，可通过 @std/... 直接加载。
// 本示例保持无网络、短时间、可确定退出，适合作为 API 参考和回归样例。

let fs = require("@std/fs");
let path = require("@std/path");
let os = require("@std/os");
let process = require("@std/process");
let crypto = require("@std/crypto");
let buffer = require("@std/buffer");
let stream = require("@std/stream");
let sse = require("@std/sse");
let events = require("@std/events");
let schema = require("@std/schema");
let toml = require("@std/toml");
let yaml = require("@std/yaml");
let xml = require("@std/xml");
let url = require("@std/url");
let timers = require("@std/timers");

function section(name) {
  console.log("");
  console.log("=== " + name + " ===");
}

function main() {
  let root = path.join(os.tmpdir(), "goscript-stdlib-cookbook-" + crypto.randomUUID());
  fs.rmSync(root, { recursive: true, force: true });
  fs.mkdirSync(root, { recursive: true });

  try {
    section("fs / path / os / process");

    let note = path.join(root, "notes", "hello.txt");
    fs.mkdirSync(path.dirname(note), { recursive: true });
    fs.writeFileAtomicSync(note, "hello native stdlib" + os.eol);
    fs.appendFileSync(note, "cwd=" + process.cwd() + os.eol);

    let copied = path.join(root, "notes", "hello.copy.txt");
    fs.copyFileSync(note, copied);

    let parsed = path.parse(copied);
    let stat = fs.statSync(copied);
    let entries = fs.readdirSync(path.dirname(copied));
    let walked = fs.walkSync(root, { includeDirs: false });

    console.log("tmp:", root);
    console.log("file:", parsed.base, "size:", stat.size, "isFile:", stat.isFile());
    console.log("entries:", entries.join(", "));
    console.log("walked files:", walked.length);
    console.log("platform:", os.platform, os.arch, "pid:", process.pid);

    section("buffer / crypto");

    let digest = crypto.sha256(fs.readTextSync(copied)).slice(0, 12);
    let bytes = buffer.from("GoScript", "utf8");
    let encoded = bytes.toString("base64");
    let decoded = buffer.from(encoded, "base64").toString("utf8");
    let filled = buffer.alloc(4, 65).toString("utf8");

    console.log("sha256 prefix:", digest);
    console.log("base64:", encoded, "decoded:", decoded);
    console.log("filled:", filled, "isBuffer:", buffer.isBuffer(bytes));

    section("toml / yaml / xml");

    let tomlFile = path.join(root, "config.toml");
    let yamlFile = path.join(root, "config.yaml");
    let xmlFile = path.join(root, "config.xml");

    fs.writeTextSync(tomlFile, "[app]\nname = \"cookbook\"\nenabled = true\nports = [8080, 8081]\n");
    fs.writeTextSync(yamlFile, "app:\n  name: cookbook\n  enabled: true\n  ports:\n    - 8080\n    - 8081\n");
    fs.writeTextSync(xmlFile, "<app name=\"cookbook\"><enabled>true</enabled></app>");

    let tomlConfig = toml.readFileSync(tomlFile);
    let yamlConfig = yaml.readFileSync(yamlFile);
    let xmlConfig = xml.readFileSync(xmlFile);

    console.log("toml app:", tomlConfig.app.name);
    console.log("yaml ports:", yamlConfig.app.ports.join(","));
    console.log("xml node:", xmlConfig.name, xmlConfig.attributes.name, xmlConfig.children[0].text);

    section("url / schema");

    let endpoint = url.URL("/v1/search?q=goscript", "https://example.com/docs/");
    endpoint.searchParams.set("page", "1");
    endpoint.searchParams.append("tag", "stdlib");

    let fileURL = url.pathToFileURL(copied);
    let filePath = url.fileURLToPath(fileURL);
    let requestSchema = JSON.parse("{\"type\":\"object\",\"required\":[\"url\",\"headers\"],\"additionalProperties\":false,\"properties\":{\"url\":{\"type\":\"string\",\"minLength\":1},\"headers\":{\"type\":\"object\"}}}");
    let requestValue = JSON.parse("{\"url\":\"\",\"headers\":{\"Accept\":\"application/json\"}}");
    requestValue.url = endpoint.toString();
    let validation = schema.validate(requestSchema, requestValue);

    console.log("endpoint:", endpoint.toString());
    console.log("file URL roundtrip:", path.basename(filePath));
    console.log("schema valid:", validation.valid);

    section("events / stream / sse / timers");

    let emitter = events.EventEmitter();
    let eventLog = [];
    emitter.on("record", function (value) {
      eventLog.push("on:" + value);
    });
    emitter.once("record", function (value) {
      eventLog.push("once:" + value);
    });
    emitter.emit("record", "first");
    emitter.emit("record", "second");

    let readable = stream.fromString("line one\nline two\n");
    let firstLine = readable.readLine();
    let rest = readable.readAll().trim();

    let eventStream = stream.fromString("data: alpha\n\nevent: delta\ndata: beta\n\n");
    let reader = sse.reader(eventStream);
    let firstEvent = reader.next();
    let secondEvent = reader.next();

    let elapsedStart = process.hrtime();
    timers.queueMicrotask(function () {
      eventLog.push("microtask");
    });
    let elapsed = process.hrtime(elapsedStart);

    console.log("events:", eventLog.join(", "));
    console.log("stream:", firstLine + " / " + rest);
    console.log("sse:", firstEvent.type + ":" + firstEvent.data, secondEvent.type + ":" + secondEvent.data);
    console.log("hrtime tuple:", elapsed.length);

    let ok = true;
    if (!fs.existsSync(copied)) {
      ok = false;
    }
    if (parsed.ext !== ".txt") {
      ok = false;
    }
    if (!stat.isFile()) {
      ok = false;
    }
    if (decoded !== "GoScript") {
      ok = false;
    }
    if (tomlConfig.app.name !== "cookbook") {
      ok = false;
    }
    if (!yamlConfig.app.enabled) {
      ok = false;
    }
    if (xmlConfig.children[0].text !== "true") {
      ok = false;
    }
    if (!validation.valid) {
      ok = false;
    }
    if (eventLog.length < 2) {
      ok = false;
    }
    if (firstLine !== "line one") {
      ok = false;
    }
    if (secondEvent.type !== "delta") {
      ok = false;
    }

    let status = "failed";
    if (ok) {
      status = "ok";
    }

    console.log("");
    console.log("cookbook status:", status);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

main();
