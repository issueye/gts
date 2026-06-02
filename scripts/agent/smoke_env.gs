let process = require("@std/process");
let os = require("@std/os");

println("cwd:" + process.cwd());
println("platform:" + os.platform);
println("tmp:" + os.tmpdir());
