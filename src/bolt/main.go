// Public Domain (-) 2012 The Bolt Authors.
// See the Bolt UNLICENSE file for details.

package bolt

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/tav/golly/log"
	"github.com/tav/golly/optparse"
	"github.com/tav/golly/runtime"
	"github.com/tav/golly/yaml"
	"os"
	"reflect"
	"sort"
)

var (
	InvalidRegistration = errors.New("bolt: you can only register functions as tasks")
	DefaultRegistry     = Registry{}
)

type Listing [][2]string

func (l Listing) String() string {
	buf := &bytes.Buffer{}
	buf.Write([]byte("Available tasks:\n\n"))
	max := 0
	for _, keydoc := range l {
		keylen := len(keydoc[0])
		if keylen > max {
			max = keylen
		}
	}
	max += 4
	tmpl := fmt.Sprintf("    %%-%ds%%s\n", max)
	for _, keydoc := range l {
		key, doc := keydoc[0], keydoc[1]
		fmt.Fprintf(buf, tmpl, key, doc)
	}
	buf.Write([]byte{'\n'})
	return buf.String()
}

type Registry map[string]*Task

func (r Registry) Listing() (listing Listing) {
	keys := r.Keys()
	for _, key := range keys {
		listing = append(listing, [2]string{key, r[key].Doc})
	}
	return
}

func (r Registry) Keys() []string {
	keys := make([]string, len(r))
	i := 0
	for key, _ := range r {
		keys[i] = key
		i += 1
	}
	sort.StringSlice(keys).Sort()
	return keys
}

func (r Registry) Main(ctx *Context, args []string) {
	opts := optparse.Parser("Usage: bolt <command-1> <command-2> ... [options]\n", "bolt 0.1")
	opts.String([]string{"--boltfile"}, "",
		"use the Boltfile at the given path", "PATH")
	opts.String([]string{"--gen"}, "",
		"save the generated executable at the given path", "PATH")
	opts.Bool([]string{"--recompile"}, false,
		"recompile the generated executable instead of using cached builds")
	noConsoleLog := opts.BoolConfig("no-console-log", false,
		"disable logging to the console [false]")
	completions := append(ctx.completions, r.Keys()...)
	opts.Completer = optparse.ListCompleter(completions...)
	opts.Parse(args)
	if len(args) == 1 {
		listing := r.Listing()
		fmt.Print(listing)
		runtime.Exit(0)
	}
	if !*noConsoleLog {
		log.AddConsoleLogger()
	}
	buf := &bytes.Buffer{}
	yaml.NormaliseID(buf, args[1])
	cmd := buf.String()
	if task, ok := r[cmd]; ok {
		task.Func.Call([]reflect.Value{reflect.ValueOf(ctx)})
	} else {
		fmt.Printf("Task not found:\n\n\t%s\n\n", args[1])
		runtime.Exit(1)
	}
}

type Task struct {
	Doc  string
	Func reflect.Value
}

func RegisterAt(registry Registry, id, doc string, task interface{}) error {
	rv := reflect.ValueOf(task)
	rt := rv.Type()
	if rt.Kind() != reflect.Func {
		return InvalidRegistration
	}
	buf := &bytes.Buffer{}
	yaml.NormaliseID(buf, id)
	registry[buf.String()] = &Task{Doc: doc, Func: rv}
	return nil
}

func Register(id, doc string, task interface{}) error {
	return RegisterAt(DefaultRegistry, id, doc, task)
}

func Main(ctx *Context) {
	DefaultRegistry.Main(ctx, os.Args)
}
