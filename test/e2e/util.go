package e2e

import (
	"fmt"
	"strings"

	"gotest.tools/assert/cmp"
)

// ContainsAll is a comparison utility, compares given substrings against
// target string and returns the gotest.tools/assert/cmp.Comaprison function.
// Provide target string as first arg, followed by any number of substring as args
func ContainsAll(target string, substrings ...string) cmp.Comparison {
	return func() cmp.Result {
		var missing []string
		for _, sub := range substrings {
			if !strings.Contains(target, sub) {
				missing = append(missing, sub)
			}
		}
		if len(missing) > 0 {
			return cmp.ResultFailure(fmt.Sprintf("\nActual output: %s\nMissing strings: %s", target, strings.Join(missing[:], ", ")))
		}
		return cmp.ResultSuccess
	}
}
