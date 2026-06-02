import fallback, { one, add } from "./math.gs";
import * as math from "./math.gs";

println("one + 2 =", add(one, 2));
println("default =", fallback);
println("namespace =", math.add(math.one, fallback));
