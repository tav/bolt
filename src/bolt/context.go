// Public Domain (-) 2012 The Bolt Authors.
// See the Bolt UNLICENSE file for details.

package bolt

import (
	"fmt"
)

type Context struct {
}

func (ctx *Context) Env() {
	fmt.Println("Env")
}
