package wrapper

import (
	"context"
	"testing"

	"github.com/hadi77ir/go-query/executors/memory"
	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data structure
type User struct {
	ID       int
	Name     string
	Email    string
	Password string
	SSN      string
	Balance  float64
}

func getTestUsers() []User {
	return []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Password: "secret1", SSN: "123-45-6789", Balance: 100.0},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Password: "secret2", SSN: "987-65-4321", Balance: 200.0},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", Password: "secret3", SSN: "111-22-3333", Balance: 300.0},
	}
}

func TestWrapperExecutor_BasicFunctionality(t *testing.T) {
	data := getTestUsers()
	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"

	// Inner executor allows: name, email, id, balance
	opts.AllowedFields = []string{"name", "email", "id", "balance"}
	innerExecutor := memory.NewExecutor(data, opts)

	// Wrapper executor allows: name, email (subset of inner)
	wrapperExecutor := NewExecutor(innerExecutor, []string{"name", "email"})

	ctx := context.Background()

	t.Run("allowed field passes", func(t *testing.T) {
		p, err := parser.NewParser("name = Alice")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		result, err := wrapperExecutor.Execute(ctx, q, &users)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalItems)
		assert.Equal(t, 1, result.ItemsReturned)
		assert.Equal(t, "Alice", users[0].Name)
	})

	t.Run("field not in wrapper list is rejected", func(t *testing.T) {
		p, err := parser.NewParser("id = 1")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		_, err = wrapperExecutor.Execute(ctx, q, &users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'id': field not allowed")
	})

	t.Run("sensitive field not in wrapper list is rejected", func(t *testing.T) {
		p, err := parser.NewParser("password = secret1")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		_, err = wrapperExecutor.Execute(ctx, q, &users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'password': field not allowed")
	})

	t.Run("field not in inner executor list is rejected by inner executor", func(t *testing.T) {
		// Create wrapper that allows password, but inner executor doesn't
		wrapperWithPassword := NewExecutor(innerExecutor, []string{"name", "email", "password"})

		p, err := parser.NewParser("password = secret1")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		_, err = wrapperWithPassword.Execute(ctx, q, &users)
		require.Error(t, err)
		// Inner executor will reject it because password is not in inner executor's allowed list
		assert.Contains(t, err.Error(), "field not allowed")
	})
}

func TestWrapperExecutor_EmptyAllowedFields(t *testing.T) {
	data := getTestUsers()
	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	opts.AllowedFields = []string{"name", "email", "id"}

	innerExecutor := memory.NewExecutor(data, opts)

	// Wrapper with empty allowed fields means no restriction from wrapper
	wrapperExecutor := NewExecutor(innerExecutor, []string{})

	ctx := context.Background()

	t.Run("empty wrapper list allows all fields that inner executor allows", func(t *testing.T) {
		p, err := parser.NewParser("name = Alice")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		result, err := wrapperExecutor.Execute(ctx, q, &users)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("empty wrapper list still respects inner executor restrictions", func(t *testing.T) {
		p, err := parser.NewParser("password = secret1")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		_, err = wrapperExecutor.Execute(ctx, q, &users)
		require.Error(t, err)
		// Inner executor will reject it
		assert.Contains(t, err.Error(), "field not allowed")
	})
}

func TestWrapperExecutor_SortFieldRestriction(t *testing.T) {
	data := getTestUsers()
	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	opts.AllowedFields = []string{"name", "email", "id", "balance"}

	innerExecutor := memory.NewExecutor(data, opts)
	wrapperExecutor := NewExecutor(innerExecutor, []string{"name", "email"})

	ctx := context.Background()

	t.Run("sort by allowed field", func(t *testing.T) {
		p, err := parser.NewParser("name = Alice")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)
		q.SortBy = "name"

		var users []User
		result, err := wrapperExecutor.Execute(ctx, q, &users)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("sort by restricted field is rejected", func(t *testing.T) {
		p, err := parser.NewParser("name = Alice")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)
		q.SortBy = "balance"

		var users []User
		_, err = wrapperExecutor.Execute(ctx, q, &users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'balance': field not allowed")
	})
}

func TestWrapperExecutor_ComplexFilters(t *testing.T) {
	data := getTestUsers()
	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	opts.AllowedFields = []string{"name", "email", "id", "balance"}

	innerExecutor := memory.NewExecutor(data, opts)
	wrapperExecutor := NewExecutor(innerExecutor, []string{"name", "email"})

	ctx := context.Background()

	t.Run("AND with all allowed fields", func(t *testing.T) {
		p, err := parser.NewParser("name = Alice AND email = 'alice@example.com'")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		result, err := wrapperExecutor.Execute(ctx, q, &users)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("AND with one restricted field is rejected", func(t *testing.T) {
		p, err := parser.NewParser("name = Alice AND balance > 50")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		_, err = wrapperExecutor.Execute(ctx, q, &users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'balance': field not allowed")
	})

	t.Run("OR with one restricted field is rejected", func(t *testing.T) {
		p, err := parser.NewParser("name = Alice OR balance > 50")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var users []User
		_, err = wrapperExecutor.Execute(ctx, q, &users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field 'balance': field not allowed")
	})
}

func TestWrapperExecutor_Close(t *testing.T) {
	data := getTestUsers()
	opts := query.DefaultExecutorOptions()
	innerExecutor := memory.NewExecutor(data, opts)
	wrapperExecutor := NewExecutor(innerExecutor, []string{"name"})

	err := wrapperExecutor.Close()
	assert.NoError(t, err)
}

func TestWrapperExecutor_Name(t *testing.T) {
	data := getTestUsers()
	opts := query.DefaultExecutorOptions()
	innerExecutor := memory.NewExecutor(data, opts)
	wrapperExecutor := NewExecutor(innerExecutor, []string{"name"})

	assert.Equal(t, "wrapper", wrapperExecutor.Name())
}
