let fs = require("@std/fs");
let path = require("@std/path");
let crypto = require("@std/crypto");

function readText(file) {
  if (!fs.existsSync(file)) {
    return "";
  }
  return fs.readFileSync(file);
}

function appendLine(file, line) {
  let dir = path.dirname(file);
  fs.mkdirSync(dir, { recursive: true });
  fs.appendTextSync(file, line + "\n");
}

export function createJSONLSession(file) {
  let id = crypto.randomUUID();

  function append(kind, payload) {
    let record = {
      sessionId: id,
      kind: kind,
      payload: payload,
    };
    appendLine(file, JSON.stringify(record));
    return record;
  }

  function readAll() {
    let text = readText(file);
    let lines = text.split("\n");
    let records = [];
    for (let line of lines) {
      let trimmed = line.trim();
      if (trimmed !== "") {
        records.push(JSON.parse(trimmed));
      }
    }
    return records;
  }

  return {
    id: id,
    file: file,
    append: append,
    readAll: readAll,
  };
}
