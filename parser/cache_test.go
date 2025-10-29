package parser

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserCache_NoCaching(t *testing.T) {
	cache := NewParserCache(0) // No caching

	// Should parse directly
	q1, err1 := cache.Parse("name = test")
	require.NoError(t, err1)
	assert.NotNil(t, q1)

	// Should parse again (no cache, so always fresh)
	q2, err2 := cache.Parse("name = test")
	require.NoError(t, err2)
	assert.NotNil(t, q2)

	// Verify cache is empty
	assert.Equal(t, 0, cache.Size())
}

func TestParserCache_BasicCaching(t *testing.T) {
	cache := NewParserCache(10)

	// First parse should cache
	q1, err1 := cache.Parse("name = test")
	require.NoError(t, err1)
	assert.NotNil(t, q1)
	assert.Equal(t, 1, cache.Size())

	// Second parse should hit cache
	q2, err2 := cache.Parse("name = test")
	require.NoError(t, err2)
	assert.NotNil(t, q2)
	assert.Equal(t, 1, cache.Size())

	// Results should be equivalent (same filter structure)
	assert.Equal(t, q1.Filter, q2.Filter)
}

func TestParserCache_CacheEviction(t *testing.T) {
	cache := NewParserCache(3) // Small cache to test eviction

	// Add 3 entries
	cache.Parse("query1 = value1")
	cache.Parse("query2 = value2")
	cache.Parse("query3 = value3")
	assert.Equal(t, 3, cache.Size())

	// Add 4th entry - should evict one
	cache.Parse("query4 = value4")
	assert.Equal(t, 3, cache.Size())

	// Verify cache doesn't have all 4
	assert.Equal(t, 3, cache.Size())

	// Access query2 multiple times to increase its priority
	cache.Parse("query2 = value2")
	cache.Parse("query2 = value2")
	cache.Parse("query2 = value2")

	// Add new entry - query2 should be kept (high access count)
	cache.Parse("query5 = value5")
	assert.Equal(t, 3, cache.Size())

	// query2 should still be in cache (high access count)
	q, err := cache.Parse("query2 = value2")
	require.NoError(t, err)
	assert.NotNil(t, q)
}

func TestParserCache_MostUsedPrioritized(t *testing.T) {
	cache := NewParserCache(3)

	// Add 3 entries
	cache.Parse("frequent = query") // This will be accessed many times
	cache.Parse("rare1 = query")
	cache.Parse("rare2 = query")

	// Access "frequent" many times
	for i := 0; i < 20; i++ {
		cache.Parse("frequent = query")
	}

	// Add new entry - "frequent" should be kept due to high access count
	cache.Parse("new = query")

	// "frequent" should still be cached
	q, err := cache.Parse("frequent = query")
	require.NoError(t, err)
	assert.NotNil(t, q)
}

func TestParserCache_RecentlyAddedPrioritized(t *testing.T) {
	cache := NewParserCache(2)

	// Add two entries
	cache.Parse("old = query")
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	cache.Parse("new = query")

	// Add third entry - "old" should be evicted (less recent)
	cache.Parse("newest = query")
	assert.Equal(t, 2, cache.Size())

	// "new" and "newest" should be cached
	_, err1 := cache.Parse("new = query")
	assert.NoError(t, err1)

	_, err2 := cache.Parse("newest = query")
	assert.NoError(t, err2)
}

func TestParserCache_ThreadSafety(t *testing.T) {
	cache := NewParserCache(100)
	var wg sync.WaitGroup
	numGoroutines := 10
	queriesPerGoroutine := 20

	// Launch multiple goroutines that parse queries concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < queriesPerGoroutine; j++ {
				queryStr := fmt.Sprintf("query%d = value%d", id, j)
				_, err := cache.Parse(queryStr)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Cache should have entries
	assert.Greater(t, cache.Size(), 0)
	assert.LessOrEqual(t, cache.Size(), cache.maxSize)
}

func TestParserCache_Clear(t *testing.T) {
	cache := NewParserCache(10)

	// Add some entries
	cache.Parse("query1 = value1")
	cache.Parse("query2 = value2")
	assert.Equal(t, 2, cache.Size())

	// Clear cache
	cache.Clear()
	assert.Equal(t, 0, cache.Size())

	// Cache should work after clear
	q, err := cache.Parse("query3 = value3")
	require.NoError(t, err)
	assert.NotNil(t, q)
	assert.Equal(t, 1, cache.Size())
}

func TestParserCache_ErrorCaching(t *testing.T) {
	cache := NewParserCache(10)

	// Parse invalid query (should cache the error)
	_, err1 := cache.Parse("invalid query syntax !!!")
	assert.Error(t, err1)
	assert.Equal(t, 1, cache.Size())

	// Same invalid query should return cached error
	_, err2 := cache.Parse("invalid query syntax !!!")
	assert.Error(t, err2)
	assert.Equal(t, err1.Error(), err2.Error())
}

func TestParserCache_Stats(t *testing.T) {
	cache := NewParserCache(10)

	// Parse some queries
	cache.Parse("query1 = value1")
	cache.Parse("query2 = value2")
	cache.Parse("query1 = value1") // Access query1 again

	stats := cache.GetStats()
	assert.Equal(t, 2, stats.Size) // Only 2 unique queries
	assert.Greater(t, stats.TotalAccess, int64(0))
}

func TestParserCache_MixedUsage(t *testing.T) {
	cache := NewParserCache(5)

	// Add queries with different access patterns
	cache.Parse("popular = query")
	for i := 0; i < 10; i++ {
		cache.Parse("popular = query")
	}

	cache.Parse("new1 = query")
	cache.Parse("new2 = query")
	time.Sleep(10 * time.Millisecond)
	cache.Parse("new3 = query")

	// "popular" should still be cached due to high access count
	// "new" queries should be cached due to recency
	stats := cache.GetStats()
	assert.Greater(t, stats.TotalAccess, int64(10))
}

func TestParserCache_ComplexQueries(t *testing.T) {
	cache := NewParserCache(10)

	complexQueries := []string{
		"name = test AND age > 18",
		"status = active OR status = pending",
		"(category = electronics AND price < 100) OR featured = true",
		"page_size = 10 sort_by = name sort_order = desc",
		"name CONTAINS \"test\" AND tags IN [\"tag1\", \"tag2\"]",
	}

	for _, queryStr := range complexQueries {
		q, err := cache.Parse(queryStr)
		require.NoError(t, err)
		assert.NotNil(t, q)
	}

	assert.Equal(t, len(complexQueries), cache.Size())

	// All should hit cache on second access
	for _, queryStr := range complexQueries {
		q, err := cache.Parse(queryStr)
		require.NoError(t, err)
		assert.NotNil(t, q)
	}

	// Size should remain the same
	assert.Equal(t, len(complexQueries), cache.Size())
}
