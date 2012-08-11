// Public Domain (-) 2012 The Bolt Authors.
// See the Bolt UNLICENSE file for details.

package bolt

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type optFlag int

const (
	ResetEnv = optFlag(iota)
)

type Response struct {
	Stderr  *bytes.Buffer
	Stdout  *bytes.Buffer
	Success bool
	Timeout bool
}

type Settings map[string]string

type Context struct {
	completions []string
	timeout     time.Duration
	Settings    map[string]string
	EnvDict     map[string]string
}

func (ctx *Context) Chdir(path string) {
	fmt.Println("Env")
}

func (ctx *Context) Dir(path string) *Context {
	return ctx
}

func (ctx *Context) NoPrint() *Context {
	return ctx
}

func (ctx *Context) Timeout(duration time.Duration) *Context {
	return ctx
}

func (ctx *Context) Local(cmd string) *Response {
	return nil
}

func (ctx *Context) Log(msg string) {
}

func (ctx *Context) Run(cmd string) *Response {
	return nil
}

func (ctx *Context) RunScript(script string) *Response {
	return nil
}

func (ctx *Context) Sudo(cmd string) *Response {
	return nil
}

func (ctx *Context) Getenv(key string) string {
	return ctx.EnvDict[key]
}

func (ctx *Context) Setenv(key, value string, opts ...string) string {
	return ""
}

func (ctx *Context) Get(key string) string {
	return ctx.Settings[key]
}

func (ctx *Context) Set(key, value string) *Context {
	ctx.Settings[key] = value
	return ctx
}

func (ctx *Context) Autocomplete(opts ...string) {
	ctx.completions = opts
}

func (ctx *Context) GetSettings(args ...string) *Context {
	return ctx
}

func (ctx *Context) Select(s interface{}) *Context {
	switch selector := s.(type) {
	case int:
		_ = selector
	case func(Settings) Settings:
		_ = selector
	default:
		panic(fmt.Sprintf("invalid selector type %T", s))
	}
	return ctx
}

func (ctx *Context) Format(tmpl string) string {
	dict := ctx.Settings
	out := []byte{}
	i, j := 0, 0
	for len(tmpl) > 0 {
		i = strings.Index(tmpl, "#{")
		if i < 0 {
			break
		}
		out = append(out, tmpl[:i]...)
		tmpl = tmpl[i:]
		j = strings.Index(tmpl, "}")
		if j < 0 {
			break
		}
		if j > 4 && tmpl[j-2:j] == "|q" {
			out = append(out, Quote(dict[tmpl[2:j-2]])...)
		} else {
			out = append(out, dict[tmpl[2:j]]...)
		}
		tmpl = tmpl[j+1:]
	}
	out = append(out, tmpl...)
	return string(out)
}

func Quote(s string) string {
	s = strings.Replace(s, `"`, `\"`, -1)
	return `"` + s + `"`
}

func NewContext() *Context {
	return &Context{
		Settings: map[string]string{},
		EnvDict:  map[string]string{},
	}
}
