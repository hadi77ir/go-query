package parser

import (
	"sync"
	"time"

	"github.com/hadi77ir/go-query/v2/query"
)

// cacheEntry represents a cached parsed query
type cacheEntry struct {
	query       *query.Query
	err         error
	accessCount int64
	lastAccess  time.Time
	addedAt     time.Time
}

// ParserCache is a thread-safe cache for parsed queries
// It prioritizes keeping most frequently used and recently added filters
type ParserCache struct {
	mu      sync.RWMutex
	cache   map[string]*cacheEntry
	maxSize int
	now     func() time.Time // For testing
}

// NewParserCache creates a new parser cache
// maxSize: maximum number of entries to cache. 0 means no caching (all calls go directly to parser)
func NewParserCache(maxSize int) *ParserCache {
	return &ParserCache{
		cache:   make(map[string]*cacheEntry),
		maxSize: maxSize,
		now:     time.Now,
	}
}

// Parse parses the query string, checking the cache first
// If cache miss, calls the parser and stores the result
// Returns (*query.Query, error)
func (c *ParserCache) Parse(queryStr string) (*query.Query, error) {
	// If caching is disabled, parse directly
	if c.maxSize == 0 {
		return c.parseDirect(queryStr)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check cache first
	if entry, exists := c.cache[queryStr]; exists {
		// Update access statistics
		entry.accessCount++
		entry.lastAccess = c.now()
		return entry.query, entry.err
	}

	// Cache miss - parse directly
	query, err := c.parseDirect(queryStr)

	// Store in cache
	c.addToCache(queryStr, query, err)

	return query, err
}

// parseDirect parses a query string without using cache
func (c *ParserCache) parseDirect(queryStr string) (*query.Query, error) {
	parser, err := NewParser(queryStr)
	if err != nil {
		return nil, err
	}
	return parser.Parse()
}

// addToCache adds an entry to the cache, evicting if necessary
func (c *ParserCache) addToCache(queryStr string, query *query.Query, err error) {
	// If already at max size, evict one entry
	if len(c.cache) >= c.maxSize {
		c.evict()
	}

	// Add new entry
	now := c.now()
	c.cache[queryStr] = &cacheEntry{
		query:       query,
		err:         err,
		accessCount: 1,
		lastAccess:  now,
		addedAt:     now,
	}
}

// evict removes the least valuable entry from the cache
// Prioritizes keeping entries that are:
// 1. Most frequently used (higher accessCount)
// 2. Recently added (newer addedAt)
// 3. Recently accessed (newer lastAccess)
func (c *ParserCache) evict() {
	if len(c.cache) == 0 {
		return
	}

	var worstKey string
	var worstScore float64
	first := true

	now := c.now()
	for key, entry := range c.cache {
		// Calculate score: higher is better
		// Score = accessCount * weight + recency bonus
		// Recency bonus favors recently added and recently accessed entries
		ageSinceAdded := now.Sub(entry.addedAt).Seconds()
		ageSinceAccess := now.Sub(entry.lastAccess).Seconds()

		// Recency score: newer entries get higher score
		// Use inverse of age (with small epsilon to avoid division by zero)
		recencyScore := 1.0/(ageSinceAdded+1.0) + 1.0/(ageSinceAccess+1.0)

		// Combined score: access frequency weighted higher than recency
		// Access count is weighted 10x more than recency to prioritize frequently used
		score := float64(entry.accessCount)*10.0 + recencyScore

		if first || score < worstScore {
			worstScore = score
			worstKey = key
			first = false
		}
	}

	// Remove the worst entry
	if worstKey != "" {
		delete(c.cache, worstKey)
	}
}

// Clear clears all entries from the cache
func (c *ParserCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cacheEntry)
}

// Size returns the current number of cached entries
func (c *ParserCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Stats returns cache statistics
type CacheStats struct {
	Size        int
	TotalAccess int64
	HitRate     float64 // Would need to track hits/misses for accurate calculation
}

// GetStats returns current cache statistics
func (c *ParserCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalAccess int64
	for _, entry := range c.cache {
		totalAccess += entry.accessCount
	}

	return CacheStats{
		Size:        len(c.cache),
		TotalAccess: totalAccess,
	}
}
