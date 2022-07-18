package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"os"
	"path"
	"strings"
)

func pkgImports(module, root, dir string) ([]string, error) {
	pkg, err := build.ImportDir(path.Join(root, dir), 0)
	if err != nil {
		if _, nogo := err.(*build.NoGoError); nogo {
			return nil, nil
		}

		return nil, err
	}

	imports := []string{}
	for _, i := range pkg.Imports {
		if strings.HasPrefix(i, module) {
			imports = append(imports, strings.TrimPrefix(i, module))
		}
	}

	return imports, nil
}

func scanDir(module, root, dir string) (map[string][]string, error) {
	deps, err := pkgImports(module, root, dir)
	if err != nil {
		return nil, err
	}

	result := map[string][]string{dir: deps}

	entries, err := os.ReadDir(path.Join(root, dir))
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !e.IsDir() ||
			e.Name() == "vendor" ||
			e.Name() == "testdata" ||
			strings.HasPrefix(e.Name(), ".") ||
			strings.HasSuffix(e.Name(), "test") {
			continue
		}

		imports, err := scanDir(module, root, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}

		for k, v := range imports {
			if _, exists := result[k]; exists {
				return nil, fmt.Errorf("duplicate package %q in dir %q", k, dir)
			}

			result[k] = v
		}
	}

	return result, nil
}

func importMap(root string) (map[string][]string, error) {
	mod, err := os.ReadFile(path.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}

	space := bytes.IndexRune(mod, ' ')
	newline := bytes.IndexRune(mod, '\n')
	if space == -1 || newline == -1 {
		return nil, errors.New("go.mod file doesn't start with 'module <import path>'")
	}

	return scanDir(string(mod[space+1:newline])+"/", root, "")
}

func outputGraphviz(imports map[string][]string) {
	fmt.Println("digraph G {")
	for pkg, deps := range imports {
		for _, dep := range deps {
			fmt.Printf("\t\"%s\" -> \"%s\"\n", pkg, dep)
		}
	}
	fmt.Println("}")
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: godeps <path to root of go module checkout>")
		os.Exit(1)
	}

	imports, err := importMap(os.Args[1])
	if err != nil {
		fmt.Printf("Failed to compute import map: %s", err)
		os.Exit(1)
	}

	outputGraphviz(imports)
}