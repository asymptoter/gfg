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
)

var (
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

	goModuleName := getGoModuleName()
	goModPath = os.Getenv(keyGoModPath)
	baseBranchName = os.Getenv(keyBaseBranchName)

	if len(goModPath) == 0 {
		panic("ENV GO_MOD_PATH not set")
	}

	if len(baseBranchName) == 0 {
		panic("ENV BASE_BRANCH_NAME not set")
	}

	dependency := getDependency(goModuleName)

	pretty(dependency)

	toBeTestedPackages := getToBeTestedPackages(&dependency, goModuleName, baseBranchName)

	pretty(toBeTestedPackages)

	runGoTests(toBeTestedPackages)

	fmt.Printf("time elapsed: %fs\n", time.Now().Sub(t1).Seconds())
}

func runGoTests(pkgs []string) {
	for _, pkg := range pkgs {
		fmt.Printf("go test %s: ", pkg)
		cmd := exec.Command("go", "test", "--short", pkg)
		cmd.Dir = goModPath
		if err := cmd.Run(); err != nil {
			fmt.Println("failed")
			os.Exit(1)
		}
		fmt.Println("ok")
	}
}

func getDependency(goModuleName string) dependency {
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
		res = constructDependency(goModuleName)

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

func constructDependency(goModuleName string) dependency {
	packages := listPackages()

	res := dependency{
		BottomUp: make(map[string]map[string]struct{}, len(packages)),
		TopDown:  make(map[string]map[string]struct{}, len(packages)),
	}
	for _, pkg := range packages {
		res.BottomUp[pkg] = map[string]struct{}{}
		res.TopDown[pkg] = map[string]struct{}{}
	}

	for _, pkg := range packages {
		updateDependency(&res, statusNew, goModuleName, pkg)
	}

	return res
}

func updateDependency(dp *dependency, status gitFileStatus, goModuleName, pkg string) {
	switch status {
	case statusNew:
		for _, importedPackage := range getImportedPackages(pkg) {
			if strings.HasPrefix(importedPackage, goModuleName) {
				if dp.BottomUp == nil {
					dp.BottomUp = map[string]map[string]struct{}{}
				}
				if dp.BottomUp[importedPackage] == nil {
					dp.BottomUp[importedPackage] = map[string]struct{}{}
				}
				if dp.TopDown == nil {
					dp.TopDown = map[string]map[string]struct{}{}
				}
				if dp.TopDown[pkg] == nil {
					dp.TopDown[pkg] = map[string]struct{}{}
				}
				dp.BottomUp[importedPackage][pkg] = struct{}{}
				dp.TopDown[pkg][importedPackage] = struct{}{}
			}
		}
	case statusModified, statusDeleted:
		dp.TopDown[pkg] = map[string]struct{}{}
		for _, importedPackage := range getImportedPackages(pkg) {
			if strings.HasPrefix(importedPackage, goModuleName) {
				dp.TopDown[pkg][importedPackage] = struct{}{}
			}
		}
	default:
		panic(fmt.Sprintln("invalid status: ", status))
	}
}

var listPackages func() []string = func() []string {
	rcmd := "go list -buildvcs=false ./..."
	return execCommand(rcmd)
}

var getImportedPackages func(pkg string) []string = func(pkg string) []string {
	rcmd := `go list -buildvcs=false -f '{{range $imp := .Imports}}{{printf "%s\n" $imp}}{{end}}' ` + pkg
	return execCommand(rcmd)
}

var getModifiedFiles func(baseBranchName string) []string = func(baseBranchName string) []string {
	rcmd := "git --no-pager diff --name-status --relative " + baseBranchName
	return execCommand(rcmd)
}

func getGoModuleName() string {
	rcmd := "head -1 go.mod"

	// module github.com/asymptoter/gfg
	firstLine := execCommand(rcmd)[0]
	return strings.Split(firstLine, " ")[1]
}

func execCommand(rcmd string) []string {
	cmd := exec.Command("sh", "-c", rcmd)
	cmd.Dir = goModPath
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	res := strings.Split(string(output), "\n")
	return res[:len(res)-1] // Remove empty
}

func getToBeTestedPackages(dp *dependency, goModuleName, baseBranchName string) []string {
	res := []string{}
	m := map[string]struct{}{}
	for _, modifiedFile := range getModifiedFiles(baseBranchName) {
		status, partialPackagePath := parseFileName(modifiedFile)
		packagePath := goModuleName + "/" + partialPackagePath

		if _, ok := m[packagePath]; !ok {
			m[packagePath] = struct{}{}

			updateDependency(dp, status, goModuleName, packagePath)

			if !strings.Contains(packagePath, "mocks") {
				res = append(res, packagePath)
			}
		}
	}

	// Add packages that depend on modified files
	for _, d := range res {
		for pkg := range dp.BottomUp[d] {
			if _, ok := m[pkg]; !ok {
				m[pkg] = struct{}{}
				if !strings.Contains(pkg, "mocks") {
					res = append(res, pkg)
				}
			}
		}
	}

	return res
}

type gitFileStatus string

const (
	statusModified gitFileStatus = "M"
	statusNew      gitFileStatus = "A"
	statusDeleted  gitFileStatus = "D"
)

func parseFileName(path string) (gitFileStatus, string) {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return gitFileStatus(path[0]), strings.TrimSpace(path[1:i])
		}
	}
	return gitFileStatus(path[0]), ""
}

func pretty(v interface{}) {
	bs, _ := json.MarshalIndent(v, "", "    ")
	fmt.Println(string(bs))
}
