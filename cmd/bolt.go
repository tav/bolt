// Public Domain (-) 2012 The Bolt Authors.
// See the Bolt UNLICENSE file for details.

package main

import (
	"amp/fsutil"
	"amp/log"
	"amp/runtime"
	"amp/yaml"
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Task struct {
	Doc      string
	FuncName string
	ID       string
}

type Spec struct {
	FileSet *token.FileSet
	Init    bool
	Root    interface{}
	Tasks   []*Task
}

func findBoltDir(filename string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	exists, fserr := fsutil.FileExists(filepath.Join(dir, filename))
	if exists {
		return dir, nil
	}
	prev := ""
	for {
		dir = filepath.Dir(dir)
		if dir == prev {
			break
		}
		exists, _ = fsutil.FileExists(filepath.Join(dir, filename))
		if exists {
			return dir, nil
		}
		prev = dir
	}
	return "", fserr
}

func exitForParserErrors(filename string, err error) {
	if err, ok := err.(scanner.ErrorList); ok {
		for _, e := range err {
			log.Error("%s:%s", filename, e)
		}
		runtime.Exit(1)
	} else {
		runtime.StandardError(err)
	}
}

func parseBoltfile(boltpath, boltdir string) (*Spec, error) {

	// Open the Boltfile for parsing.
	boltfile, err := os.Open(boltpath)
	if err != nil {
		runtime.StandardError(err)
	}
	defer boltfile.Close()

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
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Check if the `bolt` package has been imported.
	boltImported := false
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if path == "bolt" {
			boltImported = true
			break
		}
	}

	// If not, auto-insert it.
	if !boltImported {
		impDecl := &ast.GenDecl{
			Tok: token.IMPORT,
		}
		impSpec := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"bolt"`,
			},
		}
		f.Decls = append(f.Decls, nil)
		copy(f.Decls[1:], f.Decls[:])
		f.Decls[0] = impDecl
		impDecl.Specs = append(impDecl.Specs, impSpec)
		f.Imports = append(f.Imports, impSpec)
	}

	buf := &bytes.Buffer{}
	spec := &Spec{FileSet: fileset, Root: f}

	// Find all the tasks.
	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name == "Init" {
			spec.Init = true
			continue
		}
		if funcDecl.Doc == nil {
			continue
		}
		doc := strings.TrimSpace(funcDecl.Doc.Text())
		funcDecl.Doc = nil
		if doc == "" {
			continue
		}
		task := &Task{FuncName: funcDecl.Name.Name}
		id := task.FuncName
		split := strings.SplitN(doc, " ", 2)
		if first := split[0]; len(first) >= 2 {
			idx := len(first) - 1
			if first[idx] == ':' {
				id = first[:idx]
				if len(split) > 1 {
					doc = split[1]
				} else {
					doc = ""
				}
			}
		}
		buf.Reset()
		yaml.NormaliseID(buf, id)
		task.Doc = doc
		task.ID = buf.String()
		spec.Tasks = append(spec.Tasks, task)
	}

	// Strip out comments.
	f.Comments = nil
	f.Doc = nil

	return spec, nil

}

func genExecutable(path, tempdir string, spec *Spec) error {
	f, err := os.Create(filepath.Join(tempdir, "boltfile.go"))
	if err != nil {
		return err
	}
	err = printer.Fprint(f, spec.FileSet, spec.Root)
	if err != nil {
		return err
	}
	f.Write([]byte("\nfunc main() {\n"))
	for _, task := range spec.Tasks {
		fmt.Fprintf(f, "\tbolt.Register(%q, %q, %s)\n", task.ID, task.Doc, task.FuncName)
	}
	if spec.Init {
		f.Write([]byte("\tInit()\n"))
	}
	f.Write([]byte("\tbolt.Main()\n}\n"))
	f.Close()
	err = os.Chdir(tempdir)
	if err != nil {
		return err
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command("go", "build", "-ldflags", fmt.Sprint("-o ", path), "boltfile.go")
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	err = cmd.Run()
	if err != nil {
		e := "couldn't compile boltfile.go generated from the Boltfile"
		n := true
		if stdout.Len() > 0 {
			out := strings.SplitN(stdout.String(), "\n", 2)
			if len(out) == 2 {
				e += "\n\n" + strings.TrimSpace(out[1]) + "\n"
			} else {
				e += "\n" + strings.TrimSpace(stdout.String())
			}
			n = false
		}
		if stderr.Len() > 0 {
			e += "\n" + strings.TrimSpace(stderr.String())
			n = false
		}
		if n {
			e += "\n" + err.Error()
		}
		return errors.New(e)
	}
	return nil
}

func runBoltExecutable(path, boltdir string, args []string) {
	p, err := os.StartProcess(path, args, &os.ProcAttr{
		Dir:   boltdir,
		Files: []*os.File{nil, os.Stdout, os.Stderr},
	})
	if err != nil {
		runtime.StandardError(err)
	}
	state, _ := p.Wait()
	if state.Success() {
		runtime.Exit(0)
	} else {
		runtime.Exit(1)
	}
}

func main() {

	// Setup temporary console logging.
	log.DisableConsoleTimestamp()
	log.AddConsoleLogger()

	// Set default values for command-line params.
	boltFilename := "Boltfile"
	genExecutablePath := ""

	recompile := false
	skipNext := true
	maxIdx := len(os.Args) - 1
	newArgs := []string{"bolt"}

	// Extract higher-level command-line arguments.
	for idx, arg := range os.Args {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--gen" && idx != maxIdx {
			var err error
			genExecutablePath, err = filepath.Abs(os.Args[idx+1])
			if err != nil {
				runtime.StandardError(err)
			}
			skipNext = true
		} else if arg == "--boltfile" && idx != maxIdx {
			boltFilename = os.Args[idx+1]
			skipNext = true
		} else if arg == "--recompile" {
			recompile = true
		} else {
			newArgs = append(newArgs, arg)
		}
	}

	// Try and find the directory containing the Boltfile.
	boltdir, err := findBoltDir(boltFilename)
	if err != nil {
		if _, ok := err.(*fsutil.NotFound); ok {
			log.Error("Couldn't find Boltfile")
			runtime.Exit(1)
		}
		runtime.StandardError(err)
	}

	// Generate the path to the corresponding temp directory.
	boltpath := filepath.Join(boltdir, boltFilename)
	hash := sha1.New()
	hash.Write([]byte(boltpath))
	digest := fmt.Sprintf("%x", hash.Sum(nil))
	tempdir := filepath.Join(os.TempDir(), "bolt-"+digest)

	// See if the temp directory exists and if not create it.
	exists, err := fsutil.Exists(tempdir)
	if !exists {
		if _, ok := err.(*fsutil.NotFound); !ok {
			runtime.Error("Couldn't access the temp directory: %s: %s", tempdir, err)
		}
		err = os.Mkdir(tempdir, 0744)
		if err != nil {
			runtime.Error("Couldn't create the temp directory: %s: %s", tempdir, err)
		}
	}

	// See if an up-to-date generated binary already exists and, if so, run it.
	binpath := filepath.Join(tempdir, "bolt")
	if !recompile {
		boltstat, _ := os.Stat(boltpath)
		if genExecutablePath == "" {
			binstat, err := os.Stat(binpath)
			if err == nil {
				if boltstat.ModTime().Before(binstat.ModTime()) {
					runBoltExecutable(binpath, boltdir, newArgs)
					return
				}
			}
		}
	}

	// Parse the Boltfile.
	spec, err := parseBoltfile(boltpath, boltdir)
	if err != nil {
		exitForParserErrors(boltFilename, err)
	}

	// Exit if no tasks were found.
	if len(spec.Tasks) == 0 {
		runtime.Error("No tasks were found in %s", boltpath)
	}

	// Fudge the path to the executable that needs to be generated depending on
	// whether --gen-executable was specified or not.
	genOnly := true
	if genExecutablePath == "" {
		genExecutablePath = binpath
		genOnly = false
	}

	// Generate the executable.
	err = genExecutable(genExecutablePath, tempdir, spec)
	if err != nil {
		runtime.StandardError(err)
	}

	// Exit early if --gen-executable was specified.
	if genOnly {
		log.Info("%s successfully compiled to %s", boltFilename, genExecutablePath)
		runtime.Exit(0)
	}

	// Otherwise, run the executable.
	runBoltExecutable(binpath, boltdir, newArgs)

}
