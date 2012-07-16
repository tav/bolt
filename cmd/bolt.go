// Public Domain (-) 2012 The Bolt Authors.
// See the Bolt UNLICENSE file for details.

package main

import (
	"amp/fsutil"
	"amp/log"
	"amp/runtime"
	"bytes"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Task struct {
	Name    string
	Options []string
}

func findBoltDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	exists, fserr := fsutil.FileExists(filepath.Join(dir, "Boltfile"))
	if exists {
		return dir, nil
	}
	prev := ""
	for {
		dir = filepath.Dir(dir)
		if dir == prev {
			break
		}
		exists, _ = fsutil.FileExists(filepath.Join(dir, "Boltfile"))
		if exists {
			return dir, nil
		}
		prev = dir
	}
	return "", fserr
}

func exitForParserErrors(err error) {
	if err, ok := err.(scanner.ErrorList); ok {
		for _, e := range err {
			log.Error("Boltfile:%s", e)
		}
		runtime.Exit(1)
	} else {
		runtime.StandardError(err)
	}
}

func main() {

	// Setup temporary console logging.
	log.DisableConsoleTimestamp()
	log.AddConsoleLogger()

	// Try and find the directory containing the Boltfile.
	boltdir, err := findBoltDir()
	if err != nil {
		if _, ok := err.(*fsutil.NotFound); ok {
			log.Error("Couldn't find Boltfile")
			runtime.Exit(1)
		}
		runtime.StandardError(err)
	}

	// Open the Boltfile for parsing.
	boltfile, err := os.Open(filepath.Join(boltdir, "Boltfile"))
	if err != nil {
		runtime.StandardError(err)
	}

	// Try and parse the Boltfile.
	fileset := token.NewFileSet()
	f, err := parser.ParseFile(fileset, "", boltfile, parser.ParseComments)
	if err != nil {
		// If `package main` has been omitted, auto-insert it.
		if strings.Contains(err.Error(), "expected 'package'") {
			buf := &bytes.Buffer{}
			buf.Write([]byte("package main\n"))
			boltfile.Seek(0, 0)
			io.Copy(buf, boltfile)
			fileset = token.NewFileSet()
			f, err = parser.ParseFile(fileset, "", buf, parser.ParseComments)
			if err != nil {
				exitForParserErrors(err)
			}
		} else {
			exitForParserErrors(err)
		}
	}

	// Create a temp directory for the synthetic binary.
	dir, err := ioutil.TempDir("", "bolt-")
	if err != nil {
		runtime.StandardError(err)
	}
	defer os.RemoveAll(dir)

	fmt.Println("Hello world", dir)

}
