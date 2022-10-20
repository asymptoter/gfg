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

type handler struct {
	dp           dependency
	goModuleName string
	goModDir     string
}

func main() {
	t1 := time.Now()

	h := handler{}

	h.loadGoModDir()
	h.loadGoModuleName()
	h.loadDependency()

	toBeTestedPackages := h.getToBeTestedPackages()

	h.saveDependency()

	h.runGoTests(toBeTestedPackages)

	fmt.Printf("time elapsed: %fs\n", time.Now().Sub(t1).Seconds())
}

func (h *handler) runGoTests(pkgs []string) {
	for _, pkg := range pkgs {
		fmt.Printf("go test %s: ", pkg)
		cmd := exec.Command("go", "test", "--short", pkg)
		cmd.Dir = h.goModDir
		if err := cmd.Run(); err != nil {
			fmt.Println("failed")
			os.Exit(1)
		}
		fmt.Println("ok")
	}
}

func (h *handler) loadDependency() {
	file, err := os.OpenFile(h.goModDir+"/.go_module_dependency_map", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer func() { file.Close() }()

	bs, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(bs, &h.dp); err != nil || len(h.dp.BottomUp) == 0 {
		h.constructDependency()
	}
}

func (h *handler) saveDependency() {
	file, err := os.OpenFile(h.goModDir+"/.go_module_dependency_map", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer func() { file.Close() }()

	// Store dependency map in file
	mBytes, err := json.Marshal(h.dp)
	if err != nil {
		panic(err)
	}
	if _, err := file.Write(mBytes); err != nil {
		panic(err)
	}
}

func (h *handler) constructDependency() {
	packages := listPackages(h.goModDir)

	h.dp = dependency{
		BottomUp: make(map[string]map[string]struct{}, len(packages)),
		TopDown:  make(map[string]map[string]struct{}, len(packages)),
	}

	for _, pkg := range packages {
		h.dp.BottomUp[pkg] = map[string]struct{}{}
		h.dp.TopDown[pkg] = map[string]struct{}{}
	}

	for _, pkg := range packages {
		h.updateDependency(statusNew, pkg)
	}
}

func (h *handler) updateDependency(status gitFileStatus, pkg string) {
	switch status {
	case statusNew:
		for _, importedPackage := range getImportedPackages(h.goModDir, pkg) {
			if strings.HasPrefix(importedPackage, h.goModuleName) {
				if h.dp.BottomUp == nil {
					h.dp.BottomUp = map[string]map[string]struct{}{}
				}
				if h.dp.BottomUp[importedPackage] == nil {
					h.dp.BottomUp[importedPackage] = map[string]struct{}{}
				}
				if h.dp.TopDown == nil {
					h.dp.TopDown = map[string]map[string]struct{}{}
				}
				if h.dp.TopDown[pkg] == nil {
					h.dp.TopDown[pkg] = map[string]struct{}{}
				}
				h.dp.BottomUp[importedPackage][pkg] = struct{}{}
				h.dp.TopDown[pkg][importedPackage] = struct{}{}
			}
		}
	case statusModified, statusDeleted, statusRenamed:
		h.dp.TopDown[pkg] = map[string]struct{}{}
		for _, importedPackage := range getImportedPackages(h.goModDir, pkg) {
			if strings.HasPrefix(importedPackage, h.goModuleName) {
				h.dp.TopDown[pkg][importedPackage] = struct{}{}
			}
		}
	default:
		panic(fmt.Sprintln("invalid status: ", status))
	}
}

var listPackages func(goModDir string) []string = func(goModDir string) []string {
	rcmd := "go list -buildvcs=false ./..."
	return execCommand(rcmd, goModDir)
}

var getImportedPackages func(goModDir, pkg string) []string = func(goModDir, pkg string) []string {
	rcmd := `go list -buildvcs=false -f '{{range $imp := .Imports}}{{printf "%s\n" $imp}}{{end}}' ` + pkg
	return execCommand(rcmd, goModDir)
}

var getModifiedFiles func(goModDir string) []string = func(goModDir string) []string {
	rcmd := "git --no-pager diff --name-status --relative HEAD^"
	return execCommand(rcmd, goModDir)
}

func getGitRepositoryRoot() string {
	rcmd := "git rev-parse --show-toplevel"
	return execCommand(rcmd, ".")[0]
}

func (h *handler) loadGoModDir() {
	rcmd := `find . -name "go.mod"`
	grr := getGitRepositoryRoot()
	goModPath := execCommand(rcmd, grr)[0]
	h.goModDir = strings.TrimSuffix(goModPath, "go.mod")
}

func (h *handler) loadGoModuleName() {
	rcmd := "head -1 go.mod"

	// module github.com/asymptoter/gfg
	firstLine := execCommand(rcmd, h.goModDir)[0]
	h.goModuleName = strings.Split(firstLine, " ")[1]
}

func execCommand(rcmd, goModDir string) []string {
	cmd := exec.Command("sh", "-c", rcmd)
	cmd.Dir = goModDir
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	res := strings.Split(string(output), "\n")
	return res[:len(res)-1] // Remove empty
}

func (h *handler) getToBeTestedPackages() []string {
	modifiedPackages := []string{}
	m := map[string]struct{}{}
	for _, modifiedFile := range getModifiedFiles(h.goModDir) {
		status, partialPackagePath := parseFileName(modifiedFile)
		packagePath := h.goModuleName + "/" + partialPackagePath

		if _, ok := m[packagePath]; !ok {
			m[packagePath] = struct{}{}

			h.updateDependency(status, packagePath)

			if !strings.Contains(packagePath, "mocks") {
				modifiedPackages = append(modifiedPackages, packagePath)
			}
		}
	}

	// Add packages that depend on modified files
	res := []string{}
	for _, d := range modifiedPackages {
		fmt.Println(d)
		for pkg := range h.dp.BottomUp[d] {
			fmt.Println("    " + pkg)
			if _, ok := m[pkg]; !ok {
				m[pkg] = struct{}{}
				if !strings.Contains(pkg, "mocks") {
					res = append(res, pkg)
				}
			}
		}
	}

	return append(res, modifiedPackages...)
}

type gitFileStatus string

const (
	statusModified gitFileStatus = "M"
	statusNew      gitFileStatus = "A"
	statusDeleted  gitFileStatus = "D"
	statusRenamed  gitFileStatus = "R"
)

const (
	// This is string(byte(9)), not white space.
	tab = "	"
)

func parseFileName(path string) (gitFileStatus, string) {
	ss := strings.Split(path, tab)

	status := gitFileStatus(ss[0][0])

	for i := 1; i < len(ss); i++ {
		if len(ss[i]) > 0 {
			path = ss[i]
			break
		}
	}

	res := ""
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			res = strings.TrimSpace(path[:i])
			break
		}
	}
	return status, res
}

func pretty(v interface{}) {
	bs, _ := json.MarshalIndent(v, "", "    ")
	fmt.Println(string(bs))
}
