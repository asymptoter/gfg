package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	keyGoModPath      = "GO_MOD_PATH"
	keyBaseBranchName = "BASE_BRANCH_NAME"
	keyGoModuleName   = "GO_MODULE_NAME"
)

var (
	goModuleName   = os.Getenv(keyGoModuleName)
	goModPath      = os.Getenv(keyGoModPath)
	baseBranchName = os.Getenv(keyBaseBranchName)
)

/*
package dependencys:

	A imports B
	A imports C
	A imports D
	B imports C

BottomUp: if key package changed, value packages must be tested.

	B: [A]
	C: [A, B]
	D: [A]

TopDown: if key package changed, value packages in BottomUp must be updated.

	A: [B, C, D]
	B: [C]
*/
type dependency struct {
	BottomUp map[string]map[string]struct{}
	TopDown  map[string]map[string]struct{}
}

func main() {
	t1 := time.Now()

	goModuleName = os.Getenv(keyGoModuleName)
	goModPath = os.Getenv(keyGoModPath)
	baseBranchName = os.Getenv(keyBaseBranchName)

	if len(goModuleName) == 0 {
		panic("ENV GO_MODULE_NAME not set")
	}

	if len(goModPath) == 0 {
		panic("ENV GO_MOD_PATH not set")
	}

	if len(baseBranchName) == 0 {
		panic("ENV BASE_BRANCH_NAME not set")
	}

	dependency := getDependency()

	toBeTestedPackages := getToBeTestedPackages(dependency)

	bs, _ := json.MarshalIndent(dependency, "", "    ")
	fmt.Println(string(bs))

	fmt.Println(toBeTestedPackages)
	runGoTests(toBeTestedPackages)

	fmt.Printf("time elapsed: %fs\n", time.Now().Sub(t1).Seconds())
}

func runGoTests(pkgs []string) {
	for _, pkg := range pkgs {
		fmt.Printf("go test %s: ", pkg)
		cmd := exec.Command("go", "test", pkg)
		cmd.Dir = goModPath
		if err := cmd.Run(); err != nil {
			fmt.Println("failed")
			panic(err)
		}
		fmt.Println("ok")
	}
}

func getDependency() dependency {
	file, err := os.OpenFile(goModPath+"/.go_module_dependency_map", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer func() { file.Close() }()

	bs, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var res dependency
	if err := json.Unmarshal(bs, &res); err != nil || len(res.BottomUp) == 0 {
		res = constructDependency()

		// Store dependency map in file
		mBytes, err := json.Marshal(res)
		if err != nil {
			panic(err)
		}
		if _, err := file.Write(mBytes); err != nil {
			panic(err)
		}
	}

	return res
}

func constructDependency() dependency {
	cmd := exec.Command("go", "list", "./...")
	cmd.Dir = goModPath
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	packages := strings.Split(string(output), "\n")
	packages = packages[0 : len(packages)-1] // Remove empty

	res := dependency{
		BottomUp: make(map[string]map[string]struct{}, len(packages)),
		TopDown:  make(map[string]map[string]struct{}, len(packages)),
	}
	for _, pkg := range packages {
		res.BottomUp[pkg] = map[string]struct{}{}
		res.TopDown[pkg] = map[string]struct{}{}
	}

	pretty(res)

	for _, pkg := range packages {
		updateDependency(&res, pkg)
	}

	return res
}

func updateDependency(dp *dependency, pkg string) {
	for _, importedPackage := range getImportedPackages(pkg) {
		if strings.HasPrefix(importedPackage, goModuleName) {
			dp.BottomUp[importedPackage][pkg] = struct{}{}
			dp.TopDown[pkg][importedPackage] = struct{}{}
		}
	}
}

func getImportedPackages(pkg string) []string {
	rcmd := `go list -f '{{range $imp := .Imports}}{{printf "%s\n" $imp}}{{end}}' ` + pkg
	cmd := exec.Command("bash", "-c", rcmd)
	cmd.Dir = goModPath
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	res := strings.Split(string(output), "\n")
	return res[0 : len(res)-1] // Remove empty
}

func getToBeTestedPackages(dp dependency) []string {
	cmd := exec.Command("git", "diff", "--name-only", "--relative", baseBranchName)
	cmd.Dir = goModPath
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	modifiedFiles := strings.Split(string(output), "\n")
	modifiedFiles = modifiedFiles[:len(modifiedFiles)-1]

	res := []string{}
	m := map[string]struct{}{}
	for i := range modifiedFiles {
		modifiedFiles[i] = trimFileName(modifiedFiles[i])
		pkg := goModuleName + "/" + modifiedFiles[i]
		res = append(res, pkg)
		m[pkg] = struct{}{}
	}

	for _, d := range res {
		for pkg := range dp.TopDown[d] {
			updateDependency(&dp, pkg)
		}

		for pkg := range dp.BottomUp[d] {
			if _, ok := m[pkg]; !ok {
				m[pkg] = struct{}{}
				res = append(res, pkg)
			}
		}
	}

	return res
}

func trimFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}

func pretty(v any) {
	bs, _ := json.MarshalIndent(v, "", "    ")
	fmt.Println(string(bs))
}
