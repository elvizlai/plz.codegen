package gen

import (
	"bytes"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"plugin"
	"reflect"
	"runtime"
	"sync"
)

var compilePluginMutex = &sync.Mutex{}
var isInBatchCompileMode = false

// CompilePlugin in the build time, producing a .so file
func CompilePlugin(soFileName string, compileOpTriggers ...func()) {
	compilePluginMutex.Lock()
	defer compilePluginMutex.Unlock()
	isInBatchCompileMode = true
	defer func() {
		isInBatchCompileMode = false
	}()
	compileOps := []*compileOp{}
	for i, compileOpTrigger := range compileOpTriggers {
		compileOp := collectCompileOp(compileOpTrigger)
		if compileOp == nil {
			panic(fmt.Sprintf("the %d trigger did not call any gen.Compile internally", i))
		}
		compileOps = append(compileOps, compileOp)
	}
	generator := newGenerator()
	source := ""
	for _, compileOp := range compileOps {
		_, oneSource := generator.gen(compileOp.template, compileOp.kv...)
		source += oneSource
	}
	logger.Debug("generated source", "source", source)
	compileAndOpenPlugin(soFileName, source)
}
func newGenerator() *generator {
	return &generator{
		generatedTypes:        map[reflect.Type]bool{},
		generatedFuncs:        map[string]bool{},
		generatedDeclarations: map[string]bool{},
	}
}

func collectCompileOp(compileOpTrigger func()) (op *compileOp) {
	defer func() {
		recoved := recover()
		op, _ = recoved.(*compileOp)
	}()
	compileOpTrigger()
	return nil
}

func compileAndOpenPlugin(soFileName string, source string) *plugin.Plugin {
	prelog := `
package main
import "unsafe"
import "fmt"
`
	for pkg := range ImportPackages {
		prelog += "import \"" + pkg + "\"\n"
	}
	prelog += `
var debugLog = fmt.Println

type emptyInterface struct {
	typ  unsafe.Pointer
	word unsafe.Pointer
}
	`
	source = prelog + source
	fileName := hash(source)
	homeDir := os.Getenv("HOME")
	cacheDir := homeDir + "/.wombat/"
	//cacheDir = "/tmp/"
	if _, err := os.Stat(cacheDir); err != nil {
		os.Mkdir(cacheDir, 0777)
	}
	srcFileName := cacheDir + fileName + ".go"
	if soFileName == "" {
		soFileName = cacheDir + fileName + ".so"
	}
	if _, err := os.Stat(soFileName); err != nil {
		err := ioutil.WriteFile(srcFileName, []byte(source), 0666)
		if err != nil {
			panic("failed to generate source code: " + err.Error())
		}
		logger.Debug("build plugin", "soFileName", soFileName, "srcFileName", srcFileName)
		cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", soFileName, srcFileName)
		var errBuf bytes.Buffer
		cmd.Stderr = &errBuf
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		err = cmd.Run()
		if err != nil {
			logger.Error("compile failed", "srcFileName", srcFileName, "source", annotateLines(source))
			panic("failed to compile generated plugin: " + err.Error() + ", " + errBuf.String())
		}
	}
	logger.Debug("open plugin", "soFileName", soFileName)
	thePlugin, err := plugin.Open(soFileName)
	if err != nil {
		panic("failed to load generated plugin: " + err.Error())
	}
	return thePlugin
}

func hash(source string) string {
	h := sha1.New()
	h.Write([]byte(source))
	h.Write([]byte(runtime.Version()))
	return "g" + base32.StdEncoding.EncodeToString(h.Sum(nil))
}
