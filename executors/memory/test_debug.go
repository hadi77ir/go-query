package memory

import (
	"context"
	"testing"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
)

func TestDebugSecurity(t *testing.T) {
	type User struct {
		ID       int
		Name     string
		Password string
	}

	users := []User{
		{ID: 1, Name: "Alice", Password: "secret1"},
	}

	opts := query.DefaultExecutorOptions()
	opts.AllowedFields = []string{"id", "name"}
	executor := NewExecutor(users, opts)

	p, _ := parser.NewParser("password = secret1")
	q, _ := p.Parse()

	var results []User
	_, err := executor.Execute(context.Background(), q, &results)

	t.Logf("Error: %v", err)
	t.Logf("Results: %v", results)

	if err == nil {
		t.Fatal("Expected error for querying 'password' field, got nil")
	}
}
