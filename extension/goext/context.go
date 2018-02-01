package goext

import (
	"context"
	"net/http"
)

// Context represents a context of a handler
type Context interface {
	GetID() (string, bool)
	SetID(id string) Context

	GetSchemaID() (string, bool)
	SetSchemaID(schemaID string) Context

	SetISchema(schema ISchema) Context
	GetISchema() (ISchema, bool)
	DeleteISchema()

	GetResource() (Resource, bool)
	SetResource(resource Resource) Context
	DeleteResource()

	SetTransaction(tx ITransaction) Context
	GetTransaction() (ITransaction, bool)
	DeleteTransaction()

	SetContext(context context.Context) Context
	GetContext() (context.Context, bool)
	DeleteContext()

	GetIsTopLevelHandler() (bool, bool)
	SetTopLevelHandler(isTopLevelHandler bool) Context
	DeleteIsTopLevelHandler()

	GetHttpRequest() (*http.Request, bool)
	SetHttpRequest(req *http.Request) Context

	GetTenantID() (string, bool)
	SetTenantID(tenantID string) Context

	GetResponse() (interface{}, bool)
	SetResponse(response interface{}) Context

	SetInput(input map[string]interface{}) Context
	GetInput() (map[string]interface{}, bool)

	GetAuth() (interface{}, bool)
	SetAuth(auth interface{}) Context

	GetRole() (interface{}, bool)
	SetRole(role interface{}) Context

	Clone() Context
}

type GohanContext map[string]interface{}

func (c GohanContext) GetAuth() (interface{}, bool) {
	auth, ok := c["auth"]
	return auth, ok
}

func (c GohanContext) SetAuth(auth interface{}) Context {
	c["auth"] = auth
	return c
}

func (c GohanContext) GetRole() (interface{}, bool) {
	role, ok := c["role"]
	return role, ok
}

func (c GohanContext) SetRole(role interface{}) Context {
	c["role"] = role
	return c
}

func (c GohanContext) SetInput(input map[string]interface{}) Context {
	c["input"] = input
	return c
}

func (c GohanContext) GetInput() (map[string]interface{}, bool) {
	input, ok := c["input"]
	if !ok {
		return nil, ok
	}
	if input != nil {
		return input.(map[string]interface{}), ok
	}

	return nil, ok
}

func (c GohanContext) GetID() (string, bool) {
	id, ok := c["id"]
	if !ok {
		return "", ok
	}
	return id.(string), ok
}

func (c GohanContext) SetID(id string) Context {
	c["id"] = id
	return c
}

func (c GohanContext) GetResponse() (interface{}, bool) {
	response, ok := c["response"]
	if !ok {
		return nil, ok
	}
	return response, ok
}

func (c GohanContext) SetResponse(response interface{}) Context {
	c["response"] = response
	return c
}

func (c GohanContext) GetTenantID() (string, bool) {
	tenantID, ok := c["tenant_id"]
	if !ok {
		return "", ok
	}
	return tenantID.(string), ok
}

func (c GohanContext) SetTenantID(tenantID string) Context {
	c["tenant_id"] = tenantID
	return c
}

func (c GohanContext) SetISchema(schema ISchema) Context {
	c["schema"] = schema
	return c
}

func (c GohanContext) GetISchema() (ISchema, bool) {
	schema, ok := c["schema"]
	if !ok {
		return nil, ok
	}
	if schema != nil {
		return schema.(ISchema), ok
	}

	return nil, ok
}

func (c GohanContext) DeleteISchema() {
	delete(c, "schema")
}

func (c GohanContext) Clone() Context {
	contextCopy := GohanContext{}
	for k, v := range c {
		contextCopy[k] = v
	}
	return contextCopy
}

func (c GohanContext) DeleteResource() {
	delete(c, "resource")
}

func (c GohanContext) DeleteTransaction() {
	delete(c, "transaction")
}

func (c GohanContext) DeleteContext() {
	delete(c, "context")
}

func (c GohanContext) GetIsTopLevelHandler() (bool, bool) {
	isTopLevelHandler, ok := c[KeyTopLevelHandler]
	if !ok {
		return false, ok
	}
	return isTopLevelHandler.(bool), ok
}

func (c GohanContext) SetTopLevelHandler(isTopLevelHandler bool) Context {
	c[KeyTopLevelHandler] = isTopLevelHandler
	return c
}

func (c GohanContext) DeleteIsTopLevelHandler() {
	delete(c, KeyTopLevelHandler)
}

func (c GohanContext) GetHttpRequest() (*http.Request, bool) {
	httpReq, ok := c["http_request"]
	if !ok {
		return nil, ok
	}
	if httpReq != nil {
		return httpReq.(*http.Request), ok
	}
	return nil, ok
}

func (c GohanContext) SetHttpRequest(req *http.Request) Context {
	c["http_request"] = req
	return c
}

func (c GohanContext) IsTopLevelHandler() bool {
	_, isTopLevelHandler := c[KeyTopLevelHandler]
	return isTopLevelHandler
}

func (c GohanContext) GetSchemaID() (string, bool) {
	schemaID, ok := c["schema_id"]
	if !ok {
		return "", ok
	}
	return schemaID.(string), ok
}

func (c GohanContext) SetSchemaID(schemaID string) Context {
	c["schema_id"] = schemaID
	return c
}

func (c GohanContext) GetResource() (Resource, bool) {
	resource, ok := c["resource"]
	if !ok {
		return nil, ok
	}
	return resource, ok
}

func (c GohanContext) SetResource(resource Resource) Context {
	c["resource"] = resource
	return c
}

func (c GohanContext) SetTransaction(tx ITransaction) Context {
	c["transaction"] = tx
	return c
}

func (c GohanContext) GetTransaction() (ITransaction, bool) {
	tx, ok := c["transaction"]
	if !ok {
		return nil, ok
	}
	if tx != nil {
		return tx.(ITransaction), ok
	}

	return nil, ok
}

func (c GohanContext) SetContext(context context.Context) Context {
	c["context"] = context
	return c
}

func (c GohanContext) GetContext() (context.Context, bool) {
	ctx, ok := c["context"]
	if !ok {
		return nil, ok
	}
	if ctx != nil {
		return ctx.(context.Context), ok
	}
	return nil, ok
}

func MakeContext() Context {
	return GohanContext{}
}

func GetContext(requestContext Context) context.Context {
	ctx, ok := requestContext.GetContext()
	if ok {
		return ctx
	}
	return context.Background()
}
