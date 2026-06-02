let stream = require("@std/stream");
let sse = require("@std/sse");

let body = stream.fromString("data: alpha\n\nevent: delta\ndata: beta\n\n");
let reader = sse.reader(body);

let first = reader.next();
let second = reader.next();
let end = reader.next();

let endKind = "not-end";
if (end === null) {
  endKind = "end";
}

println(first.data + ":" + second.type + ":" + second.data + ":" + endKind);
