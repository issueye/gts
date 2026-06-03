import { runDemo } from "@/agent/demo";

let result = runDemo();
println(result.answer);
println("events=" + String(result.events));

result.answer;
