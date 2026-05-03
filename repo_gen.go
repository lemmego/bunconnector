package bunconnector

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lemmego/api/app"
	"github.com/lemmego/cli"
	"github.com/lemmego/fsys"
	"github.com/spf13/cobra"
)

//go:embed repo.txt
var repoStub string

type RepoConfig struct {
	Name string
}

type RepoGenerator struct {
	name string
}

func NewRepoGenerator(rc *RepoConfig) *RepoGenerator {
	return &RepoGenerator{rc.Name}
}

func (rg *RepoGenerator) GetPackagePath() string {
	return "internal/repos"
}

func (rg *RepoGenerator) GetStub() string {
	return repoStub
}

func (rg *RepoGenerator) Generate(appendable ...[]byte) error {
	fs := fsys.NewLocalStorage("")
	parts := strings.Split(rg.GetPackagePath(), "/")
	packageName := rg.GetPackagePath()

	if len(parts) > 0 {
		packageName = parts[len(parts)-1]
	}

	modName, err := cli.GetModuleName()
	if err != nil {
		return err
	}

	tmplData := map[string]interface{}{
		"PackageName": packageName,
		"Name":        rg.name,
		"ModuleName":  modName,
	}

	if len(appendable) > 0 {
		tmplData["Appendable"] = string(appendable[0])
	}

	output, err := cli.ParseTemplate(tmplData, rg.GetStub(), cli.CommonFuncs)
	if err != nil {
		return err
	}

	fileName := rg.GetPackagePath() + "/" + rg.name + "_repo.go"
	if exists, _ := fs.Exists(rg.GetPackagePath()); exists {
		return fs.Write(fileName, []byte(output))
	}

	fs.CreateDirectory(rg.GetPackagePath())
	return fs.Write(fileName, []byte(output))
}

var genBunRepoCmd = func(a app.App) *cobra.Command {
	return &cobra.Command{
		Use:     "bun:repo",
		Aliases: []string{"bun:r"},
		Short:   "Generate a Bun repository",
		Long:    `Generate a repository with embedded GPA repository and custom methods`,
		Run: func(cmd *cobra.Command, args []string) {
			shouldRunInteractively, _ := cmd.PersistentFlags().GetBool("interactive")
			var repoName string

			if !shouldRunInteractively && len(args) == 0 {
				fmt.Println("Please provide a repository name")
				return
			}

			if shouldRunInteractively && len(args) == 0 {
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Enter the repository name in snake_case (e.g. 'user' for UserRepository)").
							Value(&repoName).
							Validate(cli.SnakeCase),
					),
				)

				if err := form.Run(); err != nil {
					fmt.Println(err)
					return
				}
			} else {
				repoName = args[0]
			}

			rg := NewRepoGenerator(&RepoConfig{Name: repoName})
			if err := rg.Generate(); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Repository generated successfully.")
		},
	}
}
