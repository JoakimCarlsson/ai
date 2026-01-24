package prompt

import (
	"hash/fnv"
	"strconv"
	"sync"
	"text/template"
)

// Cache provides thread-safe caching of parsed templates.
type Cache struct {
	templates sync.Map
}

// NewCache creates a new template cache.
func NewCache() *Cache {
	return &Cache{}
}

// Get retrieves a parsed template from cache by key.
func (c *Cache) Get(key string) *template.Template {
	if v, ok := c.templates.Load(key); ok {
		return v.(*template.Template)
	}
	return nil
}

// Set stores a parsed template in the cache.
func (c *Cache) Set(key string, t *template.Template) {
	c.templates.Store(key, t)
}

// Clear removes all cached templates.
func (c *Cache) Clear() {
	c.templates.Range(func(key, _ any) bool {
		c.templates.Delete(key)
		return true
	})
}

func hashSource(source string) string {
	h := fnv.New64a()
	h.Write([]byte(source))
	return strconv.FormatUint(h.Sum64(), 36)
}
