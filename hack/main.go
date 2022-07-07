package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"log"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

type field struct {
	name     string
	comments []string
}

func main() {
	const v1pkg = "k8s.io/api/core/v1"

	pkgs, err := packages.Load(nil, v1pkg)
	if err != nil {
		log.Fatalf("Could not import package %q: %v", v1pkg, err)
	}

	if len(pkgs) != 1 {
		log.Fatalf("Unexpected number of packages: expected 1, got %d", len(pkgs))
	}

	if len(pkgs[0].CompiledGoFiles) == 0 {
		log.Fatal("No Go files detected")
	}

	pkgDir := filepath.Dir(pkgs[0].GoFiles[0])
	fst := token.NewFileSet()

	astPkgs, err := parser.ParseDir(fst, pkgDir, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Could not parse the AST: %v", err)
	}

	astPkg := astPkgs["v1"]

	var container *ast.Object

	for f, af := range astPkg.Files {
		if o := af.Scope.Lookup("Container"); o != nil {
			log.Printf("Container found in %q", f)
			container = o
		}
	}

	if container == nil {
		log.Fatal("Container type not found")
	}

	astFieldList := container.Decl.(*ast.TypeSpec).Type.(*ast.StructType).Fields.List

	fields := make([]field, 0, len(astFieldList))

	for _, f := range astFieldList {
		
		if f.Names[0].Name == "Name" {

		}
		fmt.Println(f.Names)
	}

	const tmplTxt = `
type KMMOContainer struct {
{{- range .Fields }}
	{{ .Name }}
{{- end }}
}
`

	tmpl := template.New("kmmo_container").Parse()

	println(container)

	//for _, s := range pkgs[0].Types.Scope().Lookup("Container") {
	//	log.Println(s)
	//}

	//println(v1.AzureDataDiskCachingNone)
	//
	//v1.Container{}
	//
	//pkg, err := importer.Default().Import("v1")
	//if err != nil {
	//	log.Fatalf("could not import package %q: %v", v1pkg, err)
	//}
	//
	//names := pkg.Scope().Names()
	//
	//println(names)
	////ast.NewPackage()
	//obj := ast.NewObj(ast.Typ, "ModuleContainer")
	//
	//_ = obj
	////ast.File{}
	////obj
	////doc.New()
	////typ := reflect.TypeOf(v1.Container{})
	////_ = typ
	////println(typ)
}
