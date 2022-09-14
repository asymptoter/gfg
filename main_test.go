package main

import (
	"log"
	"reflect"
	"testing"
)

func TestConstructDependency(t *testing.T) {
	listPackages = func() []string {
		return []string{
			"a",
			"b",
			"c",
			"d",
		}
	}

	importedPackageMap := map[string][]string{
		"a": []string{
			"b",
			"c",
		},
		"b": []string{
			"c",
		},
		"c": []string{},
		"d": []string{},
	}

	getImportedPackages = func(pkg string) []string {
		return importedPackageMap[pkg]
	}

	res := constructDependency("")
	exp := dependency{
		BottomUp: map[string]map[string]struct{}{
			"a": map[string]struct{}{},
			"b": map[string]struct{}{
				"a": {},
			},
			"c": map[string]struct{}{
				"a": {},
				"b": {},
			},
			"d": map[string]struct{}{},
		},
		TopDown: map[string]map[string]struct{}{
			"a": map[string]struct{}{
				"b": {},
				"c": {},
			},
			"b": map[string]struct{}{
				"c": {},
			},
			"c": map[string]struct{}{},
			"d": map[string]struct{}{},
		},
	}
	if !reflect.DeepEqual(res, exp) {
		log.Fatal("expected", exp, "actual", res)
	}
}

func TestGetToBeTestedPackages(t *testing.T) {
	dep := dependency{
		BottomUp: map[string]map[string]struct{}{
			"gfg/a": map[string]struct{}{},
			"gfg/b": map[string]struct{}{
				"gfg/a": {},
			},
			"gfg/c": map[string]struct{}{
				"gfg/a": {},
				"gfg/b": {},
			},
			"gfg/d": map[string]struct{}{},
		},
		TopDown: map[string]map[string]struct{}{
			"gfg/a": map[string]struct{}{
				"gfg/b": {},
				"gfg/c": {},
			},
			"gfg/b": map[string]struct{}{
				"gfg/c": {},
			},
			"gfg/c": map[string]struct{}{},
			"gfg/d": map[string]struct{}{},
		},
	}

	importedPackageMap := map[string][]string{
		"gfg/a": []string{
			"gfg/b",
			"gfg/c",
		},
		"gfg/b": []string{
			"gfg/c",
		},
		"gfg/c": []string{},
		"gfg/d": []string{},
		"gfg/e": []string{
			"gfg/c",
		},
	}

	getImportedPackages = func(pkg string) []string {
		return importedPackageMap[pkg]
	}

	getModifiedFiles = func(string) []string {
		return []string{
			"M  a/1.go",
			"A  e/2.go",
			"D  b/3.go",
			"R073 c/4.go",
		}
	}

	res := getToBeTestedPackages(&dep, "gfg", "")
	exp := []string{
		"gfg/a",
		"gfg/e",
		"gfg/b",
		"gfg/c",
	}
	if !reflect.DeepEqual(res, exp) {
		log.Fatal("\r\nexpected: ", exp, "\r\nactual: ", res, "\r\n")
	}
}
