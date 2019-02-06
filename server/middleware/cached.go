package middleware

import (
	"time"

	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/patrickmn/go-cache"
)

type CachedIdentityService struct {
	inner IdentityService
	cache *cache.Cache
}

func (c *CachedIdentityService) GetTenantID(tenantName string) (string, error) {
	return c.inner.GetTenantID(tenantName)
}

func (c *CachedIdentityService) GetTenantName(tenantID string) (string, error) {
	i, ok := c.cache.Get(tenantID)
	if ok {
		c.updateCounter(1, "name.hit")
		return i.(string), nil
	}
	c.updateCounter(1, "name.miss")

	name, err := c.inner.GetTenantName(tenantID)
	if err != nil {
		return "", err
	}
	c.cache.Set(tenantID, name, cache.DefaultExpiration)
	return name, nil
}

func (c *CachedIdentityService) VerifyToken(token string) (schema.Authorization, error) {
	i, ok := c.cache.Get(token)
	if ok {
		c.updateCounter(1, "token.hit")
		return i.(schema.Authorization), nil
	}
	c.updateCounter(1, "token.miss")

	a, err := c.inner.VerifyToken(token)
	if err != nil {
		return nil, err
	}
	c.cache.Set(token, a, cache.DefaultExpiration)
	return a, nil
}

func (c *CachedIdentityService) GetServiceAuthorization() (schema.Authorization, error) {
	return c.VerifyToken(c.GetServiceTokenID())
}

func (c *CachedIdentityService) GetServiceTokenID() string {
	return c.inner.GetServiceTokenID()
}

func (c *CachedIdentityService) updateCounter(delta int64, action string) {
	metrics.UpdateCounter(delta, "auth.cache.%s", action)
}

func NewCachedIdentityService(inner IdentityService, ttl time.Duration) IdentityService {
	cleanupInterval := 4 * ttl
	return &CachedIdentityService{
		inner: inner,
		cache: cache.New(ttl, cleanupInterval),
	}
}
