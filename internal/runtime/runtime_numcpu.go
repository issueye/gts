package runtime

import stdruntime "runtime"

// goruntimeNumCPU returns the number of logical CPUs usable by the process.
// It is isolated in its own file so the rest of the package can use the
// unqualified identifier "runtime" to refer to itself without shadowing the
// standard library package.
func goruntimeNumCPU() int { return stdruntime.NumCPU() }
