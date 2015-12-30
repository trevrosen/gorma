package gorma

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raphael/goa/design"
	"github.com/raphael/goa/goagen/codegen"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Generator is the application code generator.
type Generator struct {
	genfiles []string
}

// Generate is the generator entry point called by the meta generator.
func Generate(api *design.APIDefinition) ([]string, error) {
	g, err := NewGenerator()
	if err != nil {
		return nil, err
	}
	return g.Generate(api)
}

// NewGenerator returns the application code generator.
func NewGenerator() (*Generator, error) {
	return new(Generator), nil
}

// Generate produces the generated model files
func (g *Generator) Generate(api *design.APIDefinition) ([]string, error) {

	os.MkdirAll(ModelDir(), 0755)
	app := kingpin.New("Model generator", "model generator")
	codegen.RegisterFlags(app)
	_, err := app.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}
	var outPkg string
	// going to hell for this == HELP Wanted (windows) TODO:(BJK)
	outPkg = codegen.DesignPackagePath[0:strings.LastIndex(codegen.DesignPackagePath, "/")]
	if err != nil {
		panic(err)
	}
	outPkg = strings.TrimPrefix(outPkg, "src/")
	appPkg := filepath.Join(outPkg, "app")
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport(appPkg),
		codegen.SimpleImport("github.com/jinzhu/gorm"),
		codegen.SimpleImport("github.com/jinzhu/copier"),
		codegen.SimpleImport("time"),
	}

	rbacimports := []*codegen.ImportSpec{
		codegen.SimpleImport(appPkg),
		codegen.SimpleImport("github.com/mikespook/gorbac"),
	}

	rbactitle := fmt.Sprintf("%s: RBAC", api.Name)
	_, dorbac := api.Metadata["github.com/bketelsen/gorma#rbac"]
	_, cached := api.Metadata["github.com/bketelsen/gorma#cached"]
	if cached {
		imports = []*codegen.ImportSpec{
			codegen.SimpleImport(appPkg),
			codegen.SimpleImport("github.com/jinzhu/gorm"),
			codegen.SimpleImport("github.com/jinzhu/copier"),
			codegen.SimpleImport("time"),
			codegen.SimpleImport("github.com/patrickmn/go-cache"),
		}
	}

	err = api.IterateUserTypes(func(res *design.UserTypeDefinition) error {
		if res.Type.IsObject() {
			title := fmt.Sprintf("%s: Models", api.Name)
			modelname := strings.ToLower(DeModel(res.TypeName))

			filename := filepath.Join(ModelDir(), modelname+"_genmodel.go")
			os.Remove(filename)
			mtw, err := NewModelWriter(filename)
			if err != nil {
				panic(err)
			}
			mtw.WriteHeader(title, "models", imports)
			if md, ok := res.Metadata["github.com/bketelsen/gorma"]; ok && md == "Model" {
				err = mtw.Execute(res)
				if err != nil {
					fmt.Println("Error executing Gorma: ", err.Error())
					g.Cleanup()
					return err
				}
			}
			if err := mtw.FormatCode(); err != nil {
				fmt.Println("Error executing Gorma: ", err.Error())
				g.Cleanup()

			}
			if err == nil {
				g.genfiles = append(g.genfiles, filename)
			}
			return nil
		}

		return nil

	})
	if dorbac {
		rbacfilename := filepath.Join(ModelDir(), "rbac_genmodel.go")
		os.Remove(rbacfilename)
		rbacw, err := NewRbacWriter(rbacfilename)
		if err != nil {
			fmt.Println("Error executing Gorma: ", err.Error())
			panic(err)
		}
		rbacw.WriteHeader(rbactitle, "models", rbacimports)
		err = rbacw.Execute(api)
		if err != nil {
			fmt.Println("Error executing Gorma: ", err.Error())
			g.Cleanup()
			return g.genfiles, err
		}
		if err := rbacw.FormatCode(); err != nil {
			fmt.Println("Error executing Gorma: ", err.Error())
			g.Cleanup()
			return nil, err
		}
		if err == nil {
			g.genfiles = append(g.genfiles, rbacfilename)
		}
	}

	return g.genfiles, err
}

// Cleanup removes all the files generated by this generator during the last invokation of Generate.
func (g *Generator) Cleanup() {
	for _, f := range g.genfiles {
		os.Remove(f)
	}
	g.genfiles = nil
}
