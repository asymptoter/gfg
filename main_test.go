package main

import (
	"os/exec"
	"testing"
)

func Test(t *testing.T) {
	output, err := exec.Command("pwd").Output()
	if err != nil {
		t.Fatal(err)
	}
	workingDir := string(output)
	workingDir = workingDir[:len(workingDir)-1]

	t.Setenv(keyGoModPath, workingDir+"/a")
	t.Setenv(keyBaseBranchName, "master")
	t.Setenv(keyGoModuleName, "github.com/asymptoter/gfg")

	main()
}
