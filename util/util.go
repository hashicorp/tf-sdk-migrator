package util

import (
	"fmt"
	"go/ast"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	refs "github.com/radeksimko/go-refs/parser"
)

func ReadOneOf(dir string, filenames ...string) (fullpath string, content []byte, err error) {
	for _, filename := range filenames {
		fullpath = filepath.Join(dir, filename)
		content, err = ioutil.ReadFile(fullpath)
		if err == nil {
			break
		}
	}
	return
}

func SearchLines(lines []string, search string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.Contains(lines[i], search) {
			return i
		}
	}
	return -1
}

func SearchLinesPrefix(lines []string, search string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], search) {
			return i
		}
	}
	return -1
}

func GetProviderPath(providerRepoName string) (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Printf("GOPATH is empty")
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	paths := append([]string{wd}, filepath.SplitList(gopath)...)

	for _, p := range paths {
		fullPath := filepath.Join(p, "src", providerRepoName)
		info, err := os.Stat(fullPath)

		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", fullPath)
			} else {
				return fullPath, nil
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("Could not find %s in working directory or GOPATH: %s", providerRepoName, gopath)
}

func FindImportedPackages(filePath string, packagesToFind []string) (foundPackages []string) {
	// TODO: check file exists so ParseFile doesn't panic
	f, err := refs.ParseFile(filePath)
	if err != nil {
		log.Print(err)
	}

	packages := make(map[string]bool)

	ast.Inspect(f, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		switch node.(type) {
		case *ast.ImportSpec:
			importPath := node.(*ast.ImportSpec).Path.Value

			for i := range packagesToFind {
				if packagesToFind[i] == strings.Trim(importPath, "\"") {
					packageName := packagesToFind[i]
					packages[packageName] = true
					break;
				}
			}
		}

		return true
	})

	foundPackages = make([]string, len(packages))
	for k := range packages {
		foundPackages = append(foundPackages, k)
	}

	return foundPackages
}
