// Public Domain (-) 2012 The Bolt Authors.
// See the Bolt UNLICENSE file for details.

package bolt

import (
	"testing"
)

func TestFormatting(t *testing.T) {
	ctx := NewContext()
	ctx.Set("moo", "x")
	ctx.Set("blah", `foo" bar\"`)
	t.Log(ctx.Format("foo bar '#{host}' ho ho #{blah|q}"))
}
