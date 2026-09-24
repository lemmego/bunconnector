package bunconnector

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestModelGeneratorRender(t *testing.T) {
	model := NewModelGenerator(&ModelConfig{
		Name: "user_profile",
		Fields: []*ModelField{
			{Name: "created_at", Type: "time.Time", Required: true},
			{Name: "display_name", Type: "string"},
		},
	})

	output, err := model.render()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "<no value>") {
		t.Fatal("generated model contains an unset template value")
	}
	if !strings.Contains(output, `import "time"`) {
		t.Fatal("generated model does not import time")
	}
	parseGeneratedGo(t, output)
}

func TestRepoGeneratorRender(t *testing.T) {
	repo := NewRepoGenerator(&RepoConfig{Name: "user_profile"})

	output, err := repo.render("example.com/project")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "<no value>") {
		t.Fatal("generated repository contains an unset template value")
	}
	parseGeneratedGo(t, output)

	output, err = repo.render("example.com/project", []byte("// custom method"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "// custom method") {
		t.Fatal("generated repository omitted appendable content")
	}
	parseGeneratedGo(t, output)
}

func parseGeneratedGo(t *testing.T, source string) {
	t.Helper()
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, parser.AllErrors); err != nil {
		t.Fatalf("generated Go is not valid syntax: %v\n%s", err, source)
	}
}
