package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	refs "github.com/radeksimko/go-refs/parser"
)

// Package represents the subset of `go list` output we are interested in
type Package struct {
	Dir           string // directory containing package sources
	ImportPath    string // import path of package in dir
	ImportComment string // path in import comment on package statement

	// Source files
	GoFiles []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)

	// Dependency information
	Imports     []string          // import paths used by this package
	ImportMap   map[string]string // map from source import to ImportPath (identity entries omitted)
	Deps        []string          // all (recursively) imported dependencies
	TestImports []string          // imports from TestGoFiles

	// Error information
	Incomplete bool            // this package or a dependency has an error
	Error      *PackageError   // error loading package
	DepsErrors []*PackageError // errors loading dependencies
}

type PackageError struct {
	Err string
}

func GoCmd(workDir string, args ...string) (*bytes.Buffer, string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, stderr.String(), fmt.Errorf("%q: %s", args, err)
	}

	return &stdout, stderr.String(), nil
}

func GoListPackageImports(providerPath string) (allImportPathsHash map[string]bool, e error) {
	out, _, err := GoCmd(providerPath, "list", "-json", "./...")
	if err != nil {
		return nil, err
	}

	allImportPathsHash = make(map[string]bool)

	dec := json.NewDecoder(bytes.NewReader(out.Bytes()))
	for {
		var p Package
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		for _, i := range p.Imports {
			allImportPathsHash[i] = true
		}
	}

	return allImportPathsHash, nil
}

func FilesWhichImport(providerPath string, importPath string) (files []string, e error) {

}

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

	for _, impSpec := range f.Imports {
		impPath, err := strconv.Unquote(impSpec.Path.Value)
		if err != nil {
			log.Print(err)
		}
		for i := range packagesToFind {
			if packagesToFind[i] == impPath {
				packageName := packagesToFind[i]
				packages[packageName] = true
			}
		}

	}

	foundPackages = make([]string, len(packages))
	for k := range packages {
		foundPackages = append(foundPackages, k)
	}

	return foundPackages
}
