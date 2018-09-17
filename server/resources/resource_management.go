// Copyright (C) 2015 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/metrics"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/go-sql-driver/mysql"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/twinj/uuid"
)

//ResourceProblem describes the kind of problem that occurred during resource manipulation.
type ResourceProblem int

//The possible resource problems
const (
	InternalServerError ResourceProblem = iota
	WrongQuery
	WrongData
	NotFound
	DeleteFailed
	CreateFailed
	UpdateFailed
	Unauthorized
	Forbidden
	ForeignKeyFailed

	goValidationContextKey = "go_validation"
)

// ResourceError is created when an anticipated problem has occurred during resource manipulations.
// It contains the original error, a message to the user and an integer indicating the type of the problem.
type ResourceError struct {
	error
	Message string
	Problem ResourceProblem
}

//NewResourceError returns a new resource error
func NewResourceError(err error, message string, problem ResourceProblem) ResourceError {
	return ResourceError{err, message, problem}
}

// ExtensionError is created when a problem has occurred during event handling. It contains the information
// required to reraise the javascript exception that caused this error.
type ExtensionError struct {
	error
	ExceptionInfo map[string]interface{}
}

func isForeignKeyFailed(err error) bool {
	if sqliteError, ok := err.(sqlite3.Error); ok {
		if sqliteError.Code == sqlite3.ErrConstraint && sqliteError.ExtendedCode == sqlite3.ErrConstraintForeignKey {
			return true
		}
	}
	if mysqlError, ok := err.(*mysql.MySQLError); ok {
		if mysqlError.Number == 1452 {
			return true
		}
	}
	return false
}

func handleForeignKeyError(err error, dataMap map[string]interface{}) error {
	log.Info("Foreign key constrain failed: %s", err.Error())
	jsonData, _ := json.Marshal(dataMap)
	return ResourceError{
		err,
		fmt.Sprintf("Related resource does not exist. Please check your request: %s", string(jsonData)),
		ForeignKeyFailed}
}

func MeasureRequestTime(timeStarted time.Time, requestType string, schemaID string) {
	metrics.UpdateTimer(timeStarted, "req.%s.%s", schemaID, requestType)
}

//resourceTransactionWithContext executes function in the db transaction and set it to the context
func resourceTransactionWithContext(ctx middleware.Context, dataStore db.DB, level transaction.Type, fn func() error) error {
	if ctx["transaction"] != nil {
		return fmt.Errorf("cannot create nested transaction")
	}

	// note:
	// context must stay the same for each retried transaction
	// so it is stored in a temporary variable and restored before each iteration
	originalCtx := middleware.Context{}

	for k, v := range ctx {
		originalCtx[k] = v
	}

	return db.WithinTx(
		dataStore,
		func(tx transaction.Transaction) error {
			for k := range ctx {
				delete(ctx, k)
			}

			for k, v := range originalCtx {
				ctx[k] = v
			}

			ctx["transaction"] = tx
			defer delete(ctx, "transaction")

			return fn()
		},
		transaction.Context(mustGetContext(ctx)),
		transaction.TraceId(traceIdOrEmpty(ctx)),
		transaction.IsolationLevel(level))
}

// ApplyPolicyForResources applies policy filtering for response
func ApplyPolicyForResources(context middleware.Context, resourceSchema *schema.Schema) error {
	policy := context["policy"].(*schema.Policy)
	rawResponse, ok := context["response"]
	if !ok {
		return fmt.Errorf("No response")
	}
	response, ok := rawResponse.(map[string]interface{})
	if !ok {
		return fmt.Errorf("extension returned invalid JSON: %v", rawResponse)
	}
	resources, ok := response[resourceSchema.Plural].([]interface{})
	if !ok {
		return nil
	}
	data := []interface{}{}
	currCond := policy.GetCurrentResourceCondition()
	for _, resource := range resources {
		resourceMap := resource.(map[string]interface{})
		if err := currCond.ApplyPropertyConditionFilter(schema.ActionRead, resourceMap, nil); err != nil {
			continue
		}
		data = append(data, policy.RemoveHiddenProperty(resourceMap))
	}
	response[resourceSchema.Plural] = data
	return nil
}

// ApplyPolicyForResource applies policy filtering for response
func ApplyPolicyForResource(context middleware.Context, resourceSchema *schema.Schema) error {
	policy := context["policy"].(*schema.Policy)
	rawResponse, ok := context["response"]
	if !ok {
		return fmt.Errorf("No response")
	}
	response, ok := rawResponse.(map[string]interface{})
	if !ok {
		return fmt.Errorf("extension returned invalid JSON: %v", rawResponse)
	}
	resource, ok := response[resourceSchema.Singular]
	if !ok {
		return nil
	}
	resourceMap := resource.(map[string]interface{})
	currCond := policy.GetCurrentResourceCondition()
	if err := currCond.ApplyPropertyConditionFilter(schema.ActionRead, resourceMap, nil); err != nil {
		return err
	}
	response[resourceSchema.Singular] = policy.RemoveHiddenProperty(resourceMap)

	return nil
}

func ValidateAttachmentsForResource(context middleware.Context, resourceSchema *schema.Schema, resourceMap map[string]interface{}) error {
	attachPolicies, hasAttachPolicies := context["attach_policies"].([]*schema.Policy)
	auth, hasAuth := context["auth"].(schema.Authorization)
	if !hasAttachPolicies || !hasAuth {
		return nil
	}

	for _, attachPolicy := range attachPolicies {
		if err := validateAttachment(context, attachPolicy, auth, resourceSchema, resourceMap); err != nil {
			return err
		}
	}
	return nil
}

//GetResources returns specified resources without calling non in_transaction events
func GetResources(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema, filter map[string]interface{}, paginator *pagination.Paginator) error {
	defer MeasureRequestTime(time.Now(), "get.resources", resourceSchema.ID)
	return resourceTransactionWithContext(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionRead),
		func() error {
			return GetResourcesInTransaction(context, resourceSchema, filter, paginator)
		},
	)
}

//GetResourcesInTransaction returns specified resources without calling non in_transaction events
func GetResourcesInTransaction(context middleware.Context, resourceSchema *schema.Schema, filter map[string]interface{}, paginator *pagination.Paginator) error {
	defer MeasureRequestTime(time.Now(), "get.resources.in_tx", resourceSchema.ID)
	mainTransaction := context["transaction"].(transaction.Transaction)
	response := map[string]interface{}{}

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("no environment for schema")
	}

	if err := extension.HandleEvent(context, environment, "pre_list_in_transaction", resourceSchema.ID); err != nil {
		return err
	}

	var o *transaction.ViewOptions
	r, ok := context["http_request"].(*http.Request)
	if ok {
		o = listOptionsFromQueryParameter(r.URL.Query())
	}
	list, total, err := mainTransaction.List(
		mustGetContext(context),
		resourceSchema,
		filter,
		o,
		paginator,
	)
	if err != nil {
		response[resourceSchema.Plural] = []interface{}{}
		context["response"] = response
		return err
	}

	data := []interface{}{}
	for _, resource := range list {
		data = append(data, resource.Data())
	}
	response[resourceSchema.Plural] = data

	context["response"] = response
	context["total"] = total

	if err := extension.HandleEvent(context, environment, "post_list_in_transaction", resourceSchema.ID); err != nil {
		return err
	}
	return nil
}

//FilterFromQueryParameter makes list filter from query.
func FilterFromQueryParameter(resourceSchema *schema.Schema, queryParameters map[string][]string) transaction.Filter {
	filter := transaction.Filter{}
	for key, value := range queryParameters {
		if _, err := resourceSchema.GetPropertyByID(key); err != nil {
			log.Debug("Resource '%s' does not have %q property, ignoring filter", resourceSchema.ID, key)
		} else {
			filter[key] = value
		}
	}
	return filter
}

func listOptionsFromQueryParameter(v url.Values) *transaction.ViewOptions {
	return &transaction.ViewOptions{
		Details: parseBool(v.Get("_details"), true),
		Fields:  v["_fields"],
	}
}

func parseBool(s string, d bool) bool {
	if s == "" {
		return d
	}

	b, err := strconv.ParseBool(s)
	if err != nil {
		return d
	}

	return b
}

// GetMultipleResources returns all resources specified by the schema and query parameters
func GetMultipleResources(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema, queryParameters map[string][]string) error {
	defer MeasureRequestTime(time.Now(), "get.resources.multiple", resourceSchema.ID)
	log.Debug("Start get multiple resources!!")
	auth := context["auth"].(schema.Authorization)
	policy, err := LoadPolicy(context, "read", resourceSchema.GetPluralURL(), auth)
	if err != nil {
		return err
	}

	currCond := policy.GetCurrentResourceCondition()

	filter := FilterFromQueryParameter(resourceSchema, queryParameters)
	extendFilterByTenantAndDomain(resourceSchema, filter, schema.ActionRead, currCond, auth)
	filter = policy.RemoveHiddenProperty(filter)
	currCond.AddCustomFilters(resourceSchema, filter, auth)

	paginator, err := pagination.FromURLQuery(resourceSchema, queryParameters)
	if err != nil {
		return ResourceError{err, err.Error(), WrongQuery}
	}

	err = verifyQueryParams(resourceSchema, queryParameters)
	if err != nil {
		return ResourceError{err, err.Error(), WrongQuery}
	}

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	if err := extension.HandleEvent(context, environment, "pre_list", resourceSchema.ID); err != nil {
		return err
	}
	if rawResponse, ok := context["response"]; ok {
		if _, ok := rawResponse.(map[string]interface{}); ok {
			return nil
		}
		return fmt.Errorf("extension returned invalid JSON: %v", rawResponse)
	}

	if err := GetResources(context, dataStore, resourceSchema, filter, paginator); err != nil {
		return err
	}

	if err := extension.HandleEvent(context, environment, "post_list", resourceSchema.ID); err != nil {
		return err
	}

	if err := ApplyPolicyForResources(context, resourceSchema); err != nil {
		return err
	}

	return nil
}

func verifyQueryParams(resourceSchema *schema.Schema, queryParameters map[string][]string) error {
	for _, key := range resourceSchema.Properties {
		delete(queryParameters, key.ID)
	}
	delete(queryParameters, "sort_key")
	delete(queryParameters, "sort_order")
	delete(queryParameters, "limit")
	delete(queryParameters, "offset")

	delete(queryParameters, "_details")
	delete(queryParameters, "_fields")
	if len(queryParameters) > 0 {
		return fmt.Errorf("Unrecognized query parameters: %v", queryParameters)
	}
	return nil
}

// GetSingleResource returns the resource specified by the schema and ID
func GetSingleResource(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema, resourceID string) error {
	defer MeasureRequestTime(time.Now(), "get.single", resourceSchema.ID)

	context["id"] = resourceID
	auth := context["auth"].(schema.Authorization)
	policy, err := LoadPolicy(
		context,
		"read",
		strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1),
		auth,
	)
	if err != nil {
		return err
	}

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	if err := extension.HandleEvent(context, environment, "pre_show", resourceSchema.ID); err != nil {
		return err
	}
	if rawResponse, ok := context["response"]; ok {
		if _, ok := rawResponse.(map[string]interface{}); ok {
			return nil
		}
		return fmt.Errorf("extension returned invalid JSON: %v", rawResponse)
	}

	if err := resourceTransactionWithContext(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionRead),
		func() error {
			currCond := policy.GetCurrentResourceCondition()
			tenantIDs, domainIDs := currCond.GetTenantAndDomainFilters(schema.ActionRead, auth)
			return GetSingleResourceInTransaction(context, resourceSchema, resourceID, tenantIDs, domainIDs)
		},
	); err != nil {
		return err
	}

	if err := extension.HandleEvent(context, environment, "post_show", resourceSchema.ID); err != nil {
		return err
	}
	if err := ApplyPolicyForResource(context, resourceSchema); err != nil {
		return ResourceError{err, "", NotFound}
	}

	return nil
}

//GetSingleResourceInTransaction get resource in single transaction
func GetSingleResourceInTransaction(context middleware.Context, resourceSchema *schema.Schema, resourceID string, tenantIDs []string, domainIDs []string) (err error) {
	defer MeasureRequestTime(time.Now(), "get.single.in_tx", resourceSchema.ID)
	var options *transaction.ViewOptions
	r, ok := context["http_request"].(*http.Request)
	if ok {
		options = listOptionsFromQueryParameter(r.URL.Query())
	}
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("no environment for schema")
	}

	if err := extension.HandleEvent(context, environment, "pre_show_in_transaction", resourceSchema.ID); err != nil {
		return err
	}
	if rawResponse, ok := context["response"]; ok {
		if _, ok := rawResponse.(map[string]interface{}); ok {
			return nil
		}
		return fmt.Errorf("extension returned invalid JSON: %v", rawResponse)
	}
	filter := transaction.IDFilter(resourceID)
	if _, err := resourceSchema.GetPropertyByID("tenant_id"); err == nil && tenantIDs != nil {
		filter["tenant_id"] = tenantIDs
	}
	if _, err := resourceSchema.GetPropertyByID("domain_id"); err == nil && domainIDs != nil {
		filter["domain_id"] = domainIDs
	}

	auth := context["auth"].(schema.Authorization)
	policy := context["policy"].(*schema.Policy)
	currCond := policy.GetCurrentResourceCondition()
	currCond.AddCustomFilters(resourceSchema, filter, auth)

	object, err := mainTransaction.Fetch(mustGetContext(context), resourceSchema, filter, options)
	if object == nil {
		switch err {
		case transaction.ErrResourceNotFound:
			log.Info("Fetch failed: %v", err)
			return ResourceError{err, "Resource not found", NotFound}
		default:
			log.Error("Fetch failed: %v", err)
			return ResourceError{err, "Error when fetching resource", InternalServerError}
		}
	}

	response := map[string]interface{}{}
	response[resourceSchema.Singular] = object.Data()
	context["response"] = response

	if err := extension.HandleEvent(context, environment, "post_show_in_transaction", resourceSchema.ID); err != nil {
		return err
	}
	return
}

// CreateOrUpdateResource updates resource if it existed and otherwise creates it and returns true.
func CreateOrUpdateResource(
	ctx middleware.Context,
	dataStore db.DB, identityService middleware.IdentityService,
	resourceSchema *schema.Schema,
	resourceID string, dataMap map[string]interface{},
) (bool, error) {
	defer MeasureRequestTime(time.Now(), "create_or_update", resourceSchema.ID)

	auth := ctx["auth"].(schema.Authorization)

	//LoadPolicy
	policy, err := LoadPolicy(
		ctx,
		"update",
		strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1),
		auth,
	)
	if err != nil {
		return false, err
	}

	var exists bool

	if preTxErr := db.WithinTx(dataStore, func(preTransaction transaction.Transaction) error {
		exists, err = checkIfResourceExistsForTenant(mustGetContext(ctx), auth, resourceID, resourceSchema, policy, preTransaction)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		filter := transaction.IDFilter(resourceID)
		exists, err = checkIfResourceExists(mustGetContext(ctx), filter, resourceSchema, preTransaction)
		if err != nil {
			return err
		}
		if exists {
			return ResourceError{transaction.ErrResourceNotFound, "", Forbidden}
		}
		return nil
	}, transaction.Context(mustGetContext(ctx)), transaction.TraceId(traceIdOrEmpty(ctx)),
	); preTxErr != nil {
		return false, preTxErr
	}

	if !exists {
		dataMap["id"] = resourceID
		if err := CreateResource(ctx, dataStore, identityService, resourceSchema, dataMap); err != nil {
			return false, err
		}
		return true, err
	}

	return false, UpdateResource(ctx, dataStore, identityService, resourceSchema, resourceID, dataMap)
}

func checkIfResourceExistsForTenant(
	context context.Context,
	auth schema.Authorization,
	resourceID string,
	resourceSchema *schema.Schema,
	policy *schema.Policy,
	preTransaction transaction.Transaction,
) (bool, error) {
	filter := transaction.IDFilter(resourceID)

	currCond := policy.GetCurrentResourceCondition()
	extendFilterByTenantAndDomain(resourceSchema, filter, schema.ActionUpdate, currCond, auth)
	currCond.AddCustomFilters(resourceSchema, filter, auth)

	return checkIfResourceExists(context, filter, resourceSchema, preTransaction)
}

func checkIfResourceExists(context context.Context, filter transaction.Filter, resourceSchema *schema.Schema, preTransaction transaction.Transaction) (bool, error) {
	_, err := preTransaction.Fetch(context, resourceSchema, filter, nil)
	if err != nil {
		if err != transaction.ErrResourceNotFound {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// CreateResource creates the resource specified by the schema and dataMap
func CreateResource(
	context middleware.Context,
	dataStore db.DB,
	identityService middleware.IdentityService,
	resourceSchema *schema.Schema,
	dataMap map[string]interface{},
) error {
	defer MeasureRequestTime(time.Now(), "create", resourceSchema.ID)
	manager := schema.GetManager()
	// Load environment
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)

	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	auth := context["auth"].(schema.Authorization)

	//LoadPolicy
	policy, err := LoadPolicy(context, "create", resourceSchema.GetPluralURL(), auth)
	if err != nil {
		return err
	}

	_, err = resourceSchema.GetPropertyByID("tenant_id")
	if _, ok := dataMap["tenant_id"]; err == nil && !ok {
		tenantID := auth.TenantID()
		if tenantID == "" {
			err := errors.New("A non-empty tenant_id should be provided in the request")
			return ResourceError{err, err.Error(), WrongData}
		}
		dataMap["tenant_id"] = auth.TenantID()
	}

	if tenantID, ok := dataMap["tenant_id"]; ok && tenantID != nil {
		dataMap["tenant_name"] = auth.TenantName()
	}

	_, err = resourceSchema.GetPropertyByID("domain_id")
	if _, ok := dataMap["domain_id"]; err == nil && !ok {
		dataMap["domain_id"] = auth.DomainID()
	}

	if domainID, ok := dataMap["domain_id"]; ok && domainID != nil {
		dataMap["domain_name"] = auth.DomainName()
	}

	//Apply policy for api input
	err = policy.Check(schema.ActionCreate, auth, dataMap)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}
	delete(dataMap, "tenant_name")
	delete(dataMap, "domain_name")

	// apply property filter
	currCond := policy.GetCurrentResourceCondition()
	err = currCond.ApplyPropertyConditionFilter(schema.ActionCreate, dataMap, nil)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}
	context["resource"] = dataMap
	if id, ok := dataMap["id"]; !ok || id == "" {
		dataMap["id"] = uuid.NewV4().String()
	}
	context["id"] = dataMap["id"]
	context["schema_id"] = resourceSchema.ID

	if err := extension.HandleEvent(context, environment, "pre_create", resourceSchema.ID); err != nil {
		return err
	}

	if err := validate(context, &dataMap, resourceSchema.ValidateOnCreate); err != nil {
		return err
	}

	resource, err := manager.LoadResource(resourceSchema.ID, dataMap)
	if err != nil {
		return err
	}

	//Fillup default
	err = resource.PopulateDefaults()
	if err != nil {
		return err
	}

	context["resource"] = resource.Data()

	if err := resourceTransactionWithContext(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionCreate),
		func() error {
			return CreateResourceInTransaction(context, resourceSchema, resource)
		},
	); err != nil {
		return err
	}

	if err := extension.HandleEvent(context, environment, "post_create", resourceSchema.ID); err != nil {
		return err
	}

	if err := ApplyPolicyForResource(context, resourceSchema); err != nil {
		return ResourceError{err, "", Unauthorized}
	}
	return nil
}

//CreateResourceInTransaction create db resource model in transaction
func CreateResourceInTransaction(context middleware.Context, resourceSchema *schema.Schema, resource *schema.Resource) error {
	defer MeasureRequestTime(time.Now(), "create.in_tx", resource.Schema().ID)
	manager := schema.GetManager()
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	if err := extension.HandleEvent(context, environment, "pre_create_in_transaction", resourceSchema.ID); err != nil {
		return err
	}
	dataMap, ok := context["resource"].(map[string]interface{})
	if ok {
		var err error
		resource, err = manager.LoadResource(resourceSchema.ID, dataMap)
		if err != nil {
			return fmt.Errorf("Loading resource failed: %s", err)
		}
		if err := ValidateAttachmentsForResource(context, resourceSchema, dataMap); err != nil {
			return err
		}
	}
	if err := mainTransaction.Create(mustGetContext(context), resource); err != nil {
		log.Debug("%s transaction error", err)
		if isForeignKeyFailed(err) {
			return handleForeignKeyError(err, dataMap)
		}
		return ResourceError{
			err,
			fmt.Sprintf("Failed to store data in database: %v", err),
			CreateFailed}
	}

	response := map[string]interface{}{}
	response[resourceSchema.Singular] = resource.Data()
	context["response"] = response

	if err := extension.HandleEvent(context, environment, "post_create_in_transaction", resourceSchema.ID); err != nil {
		return err
	}

	return nil
}

// UpdateResource updates the resource specified by the schema and ID using the dataMap
func UpdateResource(
	context middleware.Context,
	dataStore db.DB, identityService middleware.IdentityService,
	resourceSchema *schema.Schema,
	resourceID string, dataMap map[string]interface{},
) error {
	defer MeasureRequestTime(time.Now(), "update", resourceSchema.ID)
	context["id"] = resourceID

	//load environment
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	auth := context["auth"].(schema.Authorization)

	//load policy
	policy, err := LoadPolicy(
		context,
		"update",
		strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1),
		auth,
	)
	if err != nil {
		return err
	}

	//fillup default values
	if tenantID, ok := dataMap["tenant_id"]; ok && tenantID != nil {
		dataMap["tenant_name"] = auth.TenantName()
	}
	if domainID, ok := dataMap["domain_id"]; ok && domainID != nil {
		dataMap["domain_name"] = auth.DomainName()
	}

	//check policy
	err = policy.Check(schema.ActionUpdate, auth, dataMap)
	delete(dataMap, "tenant_name")
	delete(dataMap, "domain_name")
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}
	needsDelete := false
	if _, ok := dataMap["id"]; !ok {
		dataMap["id"] = resourceID
		needsDelete = true
	}
	context["resource"] = dataMap
	if err := extension.HandleEvent(context, environment, "pre_update", resourceSchema.ID); err != nil {
		return err
	}
	if needsDelete {
		delete(dataMap, "id")
	}

	if err := resourceTransactionWithContext(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionUpdate),
		func() error {
			currCond := policy.GetCurrentResourceCondition()
			tenantIDs, domainIDs := currCond.GetTenantAndDomainFilters(schema.ActionRead, auth)
			return UpdateResourceInTransaction(context, resourceSchema, resourceID, dataMap, tenantIDs, domainIDs)
		},
	); err != nil {
		return err
	}

	if err := extension.HandleEvent(context, environment, "post_update", resourceSchema.ID); err != nil {
		return err
	}

	if err := ApplyPolicyForResource(context, resourceSchema); err != nil {
		return ResourceError{err, "", NotFound}
	}
	return nil
}

// UpdateResourceInTransaction updates resource in db in transaction
func UpdateResourceInTransaction(
	context middleware.Context,
	resourceSchema *schema.Schema, resourceID string,
	dataMap map[string]interface{},
	tenantIDs []string, domainIDs []string) error {
	defer MeasureRequestTime(time.Now(), "update.in_tx", resourceSchema.ID)

	manager := schema.GetManager()
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	filter := transaction.IDFilter(resourceID)
	if _, err := resourceSchema.GetPropertyByID("tenant_id"); err == nil && tenantIDs != nil {
		filter["tenant_id"] = tenantIDs
	}
	if _, err := resourceSchema.GetPropertyByID("domain_id"); err == nil && domainIDs != nil {
		filter["domain_id"] = domainIDs
	}

	var resource *schema.Resource
	var err error
	switch resourceSchema.GetLockingPolicy("update") {
	case schema.NoLocking:
		resource, err = mainTransaction.Fetch(mustGetContext(context), resourceSchema, filter, nil)
	case schema.LockRelatedResources:
		resource, err = mainTransaction.LockFetch(mustGetContext(context), resourceSchema, filter, schema.LockRelatedResources, nil)
	case schema.SkipRelatedResources:
		resource, err = mainTransaction.LockFetch(mustGetContext(context), resourceSchema, filter, schema.SkipRelatedResources, nil)
	}

	if err != nil {
		return ResourceError{err, err.Error(), WrongQuery}
	}

	if err := validate(context, &dataMap, resourceSchema.ValidateOnUpdate); err != nil {
		return err
	}
	policy := context["policy"].(*schema.Policy)
	// apply property filter
	currCond := policy.GetCurrentResourceCondition()
	err = currCond.ApplyPropertyConditionFilter(schema.ActionUpdate, resource.Data(), dataMap)
	if err != nil {
		return ResourceError{err, "", Unauthorized}
	}

	err = resource.Update(dataMap)
	if err != nil {
		return ResourceError{err, err.Error(), WrongData}
	}
	context["resource"] = resource.Data()

	if err := extension.HandleEvent(context, environment, "pre_update_in_transaction", resourceSchema.ID); err != nil {
		return err
	}

	dataMap, ok = context["resource"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Resource not JSON")
	}
	resource, err = manager.LoadResource(resourceSchema.ID, dataMap)
	if err != nil {
		return fmt.Errorf("Loading Resource failed: %s", err)
	}

	if err := ValidateAttachmentsForResource(context, resourceSchema, dataMap); err != nil {
		return err
	}

	err = mainTransaction.Update(mustGetContext(context), resource)
	if err != nil {
		if isForeignKeyFailed(err) {
			return handleForeignKeyError(err, dataMap)
		}
		return ResourceError{
			err,
			fmt.Sprintf("Failed to store data in database: %v", err),
			UpdateFailed,
		}
	}

	response := map[string]interface{}{}
	response[resourceSchema.Singular] = resource.Data()
	context["response"] = response

	if err := extension.HandleEvent(context, environment, "post_update_in_transaction", resourceSchema.ID); err != nil {
		return err
	}

	return nil
}

// DeleteResource deletes the resource specified by the schema and ID
func DeleteResource(ctx middleware.Context,
	dataStore db.DB,
	resourceSchema *schema.Schema,
	resourceID string,
) error {
	defer MeasureRequestTime(time.Now(), "delete", resourceSchema.ID)
	ctx["id"] = resourceID
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	var resource *schema.Resource
	var fetchErr error

	if err := extension.HandleEvent(ctx, environment, "pre_delete", resourceSchema.ID); err != nil {
		return err
	}

	if errPreTx := db.WithinTx(dataStore, func(preTransaction transaction.Transaction) error {
		resource, fetchErr = fetchResource(resourceID, resourceSchema, preTransaction, ctx)
		return fetchErr
	}, transaction.Context(mustGetContext(ctx)), transaction.TraceId(traceIdOrEmpty(ctx))); errPreTx != nil {
		return errPreTx
	}

	if resource != nil {
		ctx["resource"] = resource.Data()
	}
	if err := resourceTransactionWithContext(
		ctx, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionDelete),
		func() error {
			return DeleteResourceInTransaction(ctx, resourceSchema, resourceID)
		},
	); err != nil {
		return err
	}
	if err := extension.HandleEvent(ctx, environment, "post_delete", resourceSchema.ID); err != nil {
		return err
	}
	return nil
}

func fetchResource(resourceID string, resourceSchema *schema.Schema, tx transaction.Transaction, context middleware.Context) (*schema.Resource, error) {
	auth := context["auth"].(schema.Authorization)
	resource, fetchErr := fetchResourceForAction(schema.ActionDelete, auth, resourceID, resourceSchema, tx, context)
	if fetchErr != nil {
		switch fetchErr {
		case transaction.ErrResourceNotFound:
			_, err := fetchResourceForAction(schema.ActionRead, auth, resourceID, resourceSchema, tx, context)
			if err != nil {
				if err != transaction.ErrResourceNotFound {
					return nil, err
				}
				return nil, ResourceError{err, "Resource not found", NotFound}
			}
			// tenant cannot delete resource but can read it
			return nil, ResourceError{fetchErr, "", Forbidden}
		default:
			if _, ok := fetchErr.(ResourceError); ok {
				return nil, fetchErr
			}
			log.Error("Fetch failed: %v", fetchErr)
			return nil, ResourceError{fetchErr, "Error when fetching resource", InternalServerError}
		}
	}
	return resource, nil
}

func fetchResourceForAction(action string, auth schema.Authorization, resourceID string, resourceSchema *schema.Schema,
	tx transaction.Transaction, context middleware.Context) (*schema.Resource, error) {
	policy, err := LoadPolicy(
		context,
		action,
		strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1),
		auth,
	)
	if err != nil {
		return nil, err
	}
	currCond := policy.GetCurrentResourceCondition()
	filter := transaction.IDFilter(resourceID)
	extendFilterByTenantAndDomain(resourceSchema, filter, action, currCond, auth)
	currCond.AddCustomFilters(resourceSchema, filter, auth)
	resource, err := tx.Fetch(mustGetContext(context), resourceSchema, filter, nil)
	if err != nil {
		return nil, err
	}
	// tenant cannot delete resource but can read it
	return resource, nil

}

//DeleteResourceInTransaction deletes resources in a transaction
func DeleteResourceInTransaction(context middleware.Context, resourceSchema *schema.Schema, resourceID string) error {
	defer MeasureRequestTime(time.Now(), "delete.in_tx", resourceSchema.ID)
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	auth := context["auth"].(schema.Authorization)
	policy := context["policy"].(*schema.Policy)
	currCond := policy.GetCurrentResourceCondition()
	filter := transaction.IDFilter(resourceID)
	extendFilterByTenantAndDomain(resourceSchema, filter, schema.ActionDelete, currCond, auth)

	var resource *schema.Resource
	var err error
	switch resourceSchema.GetLockingPolicy("delete") {
	case schema.NoLocking:
		resource, err = mainTransaction.Fetch(mustGetContext(context), resourceSchema, filter, nil)
	case schema.LockRelatedResources:
		resource, err = mainTransaction.LockFetch(mustGetContext(context), resourceSchema, filter, schema.LockRelatedResources, nil)
	case schema.SkipRelatedResources:
		resource, err = mainTransaction.LockFetch(mustGetContext(context), resourceSchema, filter, schema.SkipRelatedResources, nil)
	}

	if err != nil {
		return err
	}
	if resource != nil {
		context["resource"] = resource.Data()
	}
	// apply property filter
	err = currCond.ApplyPropertyConditionFilter(schema.ActionUpdate, resource.Data(), nil)
	if err != nil {
		return ResourceError{err, "", Unauthorized}
	}

	if err := extension.HandleEvent(context, environment, "pre_delete_in_transaction", resourceSchema.ID); err != nil {
		return err
	}

	err = mainTransaction.Delete(mustGetContext(context), resourceSchema, resourceID)
	if err != nil {
		return ResourceError{err, "", DeleteFailed}
	}

	if err := extension.HandleEvent(context, environment, "post_delete_in_transaction", resourceSchema.ID); err != nil {
		return err
	}
	return nil
}

// ActionResource runs custom action on resource
func ActionResource(context middleware.Context, resourceSchema *schema.Schema,
	action schema.Action, resourceID string, data interface{},
) error {
	defer MeasureRequestTime(time.Now(), action.ID, resourceSchema.ID)
	actionSchema := action.InputSchema
	context["input"] = data
	context["id"] = resourceID

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	if actionSchema != nil {
		err := resourceSchema.Validate(actionSchema, data)
		if err != nil {
			return ResourceError{err, fmt.Sprintf("Validation error: %s", err), WrongData}
		}
	}

	err := extension.HandleEvent(context, environment, action.ID, resourceSchema.ID)
	if err != nil {
		return err
	}

	if _, ok := context["response"]; ok {
		return nil
	}

	return fmt.Errorf("no response")
}

func validateAttachment(
	context middleware.Context,
	policy *schema.Policy,
	auth schema.Authorization,
	resourceSchema *schema.Schema,
	dataMap map[string]interface{},
) error {
	relationPropertyName := policy.GetRelationPropertyName()
	if relationPropertyName == "*" {
		for i := range resourceSchema.Properties {
			prop := &resourceSchema.Properties[i]
			if err := validateAttachmentRelation(context, policy, auth, dataMap, prop); err != nil {
				return err
			}
		}
		return nil
	} else {
		prop, err := resourceSchema.GetPropertyByID(relationPropertyName)
		if err != nil {
			return err
		}
		return validateAttachmentRelation(context, policy, auth, dataMap, prop)
	}
}

func relatedResourceNotFoundErr(schemaId, resourceId string) error {
	return errors.Errorf("Related resource %s ID \"%s\" not found", schemaId, resourceId)
}

func errorNotFound(schemaId, resourceId string) error {
	return goext.NewErrorNotFound(relatedResourceNotFoundErr(schemaId, resourceId))
}

func errorBadRequest(schemaId, resourceId string) error {
	return goext.NewErrorBadRequest(relatedResourceNotFoundErr(schemaId, resourceId))
}

func extendFilterByTenantAndDomain(
	schema *schema.Schema,
	filter transaction.Filter, action string,
	rc *schema.ResourceCondition, auth schema.Authorization,
) {
	tenantFilter, domainFilter := rc.GetTenantAndDomainFilters(action, auth)
	if _, err := schema.GetPropertyByID("tenant_id"); err == nil && len(tenantFilter) > 0 {
		filter["tenant_id"] = tenantFilter
	}
	if _, err := schema.GetPropertyByID("domain_id"); err == nil && len(domainFilter) > 0 {
		filter["domain_id"] = domainFilter
	}
}

func validateAttachmentRelation(
	context middleware.Context,
	policy *schema.Policy,
	auth schema.Authorization,
	dataMap map[string]interface{},
	property *schema.Property,
) error {
	manager := schema.GetManager()
	if property.Relation == "" {
		// Not a relation, so skip
		return nil
	}

	relatedSchema, _ := manager.Schema(property.Relation)
	relatedResourceID, _ := dataMap[property.ID].(string)
	if relatedResourceID == "" {
		return nil
	}

	otherCond := policy.GetOtherResourceCondition()
	filter := transaction.IDFilter(relatedResourceID)
	extendFilterByTenantAndDomain(relatedSchema, filter, schema.ActionRead, otherCond, auth)
	otherCond.AddCustomFilters(relatedSchema, filter, auth)

	options := &transaction.ViewOptions{}

	mainTransaction := context["transaction"].(transaction.Transaction)
	relatedRes, err := mainTransaction.Fetch(mustGetContext(context), relatedSchema, filter, options)

	if err != nil {
		if policy.IsDeny() {
			return nil
		} else {
			return errorBadRequest(relatedSchema.ID, relatedResourceID)
		}
	}

	err = otherCond.ApplyPropertyConditionFilter(schema.ActionAttach, relatedRes.Data(), nil)
	if policy.IsDeny() {
		if err == nil {
			return errorBadRequest(relatedSchema.ID, relatedResourceID)
		}
	} else if err != nil {
		return errorNotFound(relatedSchema.ID, relatedResourceID)
	}
	return nil
}

func LoadPolicy(context middleware.Context, action, path string, auth schema.Authorization) (*schema.Policy, error) {
	manager := schema.GetManager()
	policy, role := manager.PolicyValidate(action, path, auth)
	if policy == nil {
		err := fmt.Errorf(fmt.Sprintf("No matching policy: %s %s", action, path))
		return nil, ResourceError{err, err.Error(), Unauthorized}
	}
	context["policy"] = policy
	context["attach_policies"] = manager.GetAttachmentPolicies(path, auth)
	context["role"] = role
	return policy, nil
}

type validateFunction func(interface{}) error

func validate(context middleware.Context, dataMap *map[string]interface{}, validate validateFunction) error {
	if _, ok := context[goValidationContextKey]; ok {
		if err := validate(dataMap); err != nil {
			return validationError(err)
		}
		copyResourceData(context, dataMap)
	} else {
		copyResourceData(context, dataMap)
		if err := validate(dataMap); err != nil {
			return validationError(err)
		}
	}
	return nil
}

func validationError(err error) error {
	return ResourceError{err, fmt.Sprintf("Validation error: %s", err), WrongData}
}

func copyResourceData(context middleware.Context, dataMap *map[string]interface{}) {
	if resourceData, ok := context["resource"].(map[string]interface{}); ok {
		*dataMap = resourceData
	}
}

func mustGetContext(requestContext middleware.Context) context.Context {
	return requestContext["context"].(context.Context)
}

func traceIdOrEmpty(requestContext middleware.Context) string {
	if traceId, hasTraceId := requestContext["trace_id"]; hasTraceId {
		return traceId.(string)
	}

	return ""
}
