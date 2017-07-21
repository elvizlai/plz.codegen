package main

import (
	"io/ioutil"
	"os"
	"fmt"
	"os/exec"
	"bytes"
)

func main() {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		reportError("GOPATH env not found")
	}
	writeCodeGeneratorMain(gopath, "github.com/v2pro/wombat/example/model")
	executeTmpCodegen(gopath + "/bin/tmp-codegen")
}

func writeCodeGeneratorMain(gopath string, pkgPath string) {
	dir := gopath+"/src/tmp-codegen"
	if _, err := os.Stat(dir); err != nil {
		err := os.Mkdir(dir, 0777)
		if err != nil {
			reportError(err.Error())
		}
	}
	ioutil.WriteFile(dir+"/main.go", []byte(fmt.Sprintf(`
package main
import _ "github.com/v2pro/wombat/example/model"
import "github.com/v2pro/wombat/generic"
func main() {
	generic.GenerateCode("%s", "%s")
}
	`, gopath, pkgPath)), 0666)
	goInstallTmpCodegen()
}

func goInstallTmpCodegen() {
	cmd := exec.Command("go", "install", "tmp-codegen")
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	if err != nil {
		fmt.Println(errBuf.String())
		fmt.Println(outBuf.String())
		reportError(err.Error())
	}
}

func executeTmpCodegen(file string) {
	cmd := exec.Command(file)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	err := cmd.Run()
	if err != nil {
		fmt.Println(errBuf.String())
		fmt.Println(outBuf.String())
		reportError(err.Error())
	}
}

func reportError(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
