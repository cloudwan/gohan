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
	"github.com/cloudwan/gohan/db/search"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goext/filter"
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

	tenantIDKey            = "tenant_id"
	domainIDKey            = "domain_id"
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
func GetResourcesInTransaction(context middleware.Context, resourceSchema *schema.Schema,
	filter map[string]interface{}, paginator *pagination.Paginator) error {
	defer MeasureRequestTime(time.Now(), "get.resources.in_tx", resourceSchema.ID)
	mainTransaction := mustGetTransaction(context)
	response := map[string]interface{}{}

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("no environment for schema")
	}

	if err := extension.HandleEvent(context, environment, "pre_list_in_transaction", resourceSchema.ID); err != nil {
		return err
	}

	o := &transaction.ViewOptions{Details: true}
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

//modify search fields to build like queries
func modifySearchFields(resourceSchema *schema.Schema,
	queryParameters map[string][]string, filter transaction.Filter) (transaction.Filter, error) {
	if searchFilters, ok := queryParameters["search_field"]; ok && resourceSchema.IsSubstringSearchEnabled() {
		for _, column := range searchFilters {
			if searchValues, ok := queryParameters[column]; ok {
				filter[column] = search.NewSearchField(searchValues[0])
			} else {
				err := fmt.Errorf("search value for `%s` not available in URL ", column)
				return nil, ResourceError{err, err.Error(), WrongQuery}
			}
		}
	}
	return filter, nil
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
func GetMultipleResources(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema,
	queryParameters map[string][]string) error {
	defer MeasureRequestTime(time.Now(), "get.resources.multiple", resourceSchema.ID)
	log.Debug("Start get multiple resources!!")
	auth := context["auth"].(schema.Authorization)
	policy, err := LoadPolicy(context, "read", resourceSchema.GetPluralURL(), auth)
	if err != nil {
		return err
	}

	currCond := policy.GetCurrentResourceCondition()

	propertiesFilter := FilterFromQueryParameter(resourceSchema, queryParameters)
	propertiesFilter, err = modifySearchFields(resourceSchema, queryParameters, propertiesFilter)
	if err != nil {
		return err
	}

	propertiesFilter = policy.RemoveHiddenProperty(propertiesFilter)
	propertiesFilter = applyAnyOfFilter(propertiesFilter, queryParameters)

	customFilters := transaction.Filter{}
	currCond.AddCustomFilters(resourceSchema, customFilters, auth)
	if len(customFilters) == 0 {
		customFilters = filter.True()
	}

	propertiesFilter = filter.And(propertiesFilter, customFilters)
	extendFilterByTenantAndDomain(resourceSchema, propertiesFilter, schema.ActionRead, currCond, auth)

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

	if err := GetResources(context, dataStore, resourceSchema, propertiesFilter, paginator); err != nil {
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

func applyAnyOfFilter(propertiesFilter map[string]interface{}, queryParameters map[string][]string) map[string]interface{} {
	filterFunc := filter.MaybeEmptyAndFilter
	if anyOf, ok := queryParameters["any_of"]; ok && len(anyOf) == 1 {
		if parseBool(queryParameters["any_of"][0], false) {
			filterFunc = filter.MaybeEmptyOrFilter
		}
	}
	filters := make([]filter.FilterElem, 0, len(propertiesFilter))
	for key, value := range propertiesFilter {
		filters = append(filters, filter.Eq(key, value))
	}
	return filterFunc(filters...)
}

func verifyQueryParams(resourceSchema *schema.Schema, queryParameters map[string][]string) error {
	for _, key := range resourceSchema.Properties {
		delete(queryParameters, key.ID)
	}
	delete(queryParameters, "sort_key")
	delete(queryParameters, "sort_order")
	delete(queryParameters, "limit")
	delete(queryParameters, "offset")

	delete(queryParameters, "search_field")
	delete(queryParameters, "any_of")

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
func GetSingleResourceInTransaction(context middleware.Context, resourceSchema *schema.Schema,
	resourceID string, tenantIDs []string, domainIDs []string) (err error) {
	defer MeasureRequestTime(time.Now(), "get.single.in_tx", resourceSchema.ID)
	options := &transaction.ViewOptions{Details: true}
	r, ok := context["http_request"].(*http.Request)
	if ok {
		options = listOptionsFromQueryParameter(r.URL.Query())
	}
	mainTransaction := mustGetTransaction(context)
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
	if _, err := resourceSchema.GetPropertyByID(tenantIDKey); err == nil && tenantIDs != nil {
		filter[tenantIDKey] = tenantIDs
	}
	if _, err := resourceSchema.GetPropertyByID(domainIDKey); err == nil && domainIDs != nil {
		filter[domainIDKey] = domainIDs
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

	var exists bool

	if preTxErr := db.WithinTx(dataStore, func(preTransaction transaction.Transaction) error {
		var err error
		filter := transaction.IDFilter(resourceID)
		exists, err = checkIfResourceExists(mustGetContext(ctx), filter, resourceSchema, preTransaction)
		if err != nil {
			return err
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
		return true, nil
	}

	return false, UpdateResource(ctx, dataStore, identityService, resourceSchema, resourceID, dataMap)
}

func getFilterFromPolicy(
	auth schema.Authorization,
	resourceID string,
	resourceSchema *schema.Schema,
	policy *schema.Policy,
	action string,
) transaction.Filter {
	filter := transaction.IDFilter(resourceID)

	currCond := policy.GetCurrentResourceCondition()
	extendFilterByTenantAndDomain(resourceSchema, filter, action, currCond, auth)
	currCond.AddCustomFilters(resourceSchema, filter, auth)

	return filter
}

func checkIfResourceExistsForPolicy(
	context context.Context,
	auth schema.Authorization,
	resourceID string,
	resourceSchema *schema.Schema,
	policy *schema.Policy,
	action string,
	preTransaction transaction.Transaction,
) (bool, error) {
	filter := getFilterFromPolicy(auth, resourceID, resourceSchema, policy, action)

	return checkIfResourceExists(context, filter, resourceSchema, preTransaction)
}

func checkIfResourceExists(context context.Context, filter transaction.Filter,
	resourceSchema *schema.Schema, preTransaction transaction.Transaction) (bool, error) {
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

	authDataMap, err := buildAuthDataMap(resourceSchema, auth, dataMap)
	if err != nil {
		return ResourceError{err, err.Error(), WrongData}
	}

	err = policy.CheckAccess(schema.ActionCreate, auth, authDataMap)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}

	err = policy.CheckPropertiesFilter(dataMap)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}

	if tenantID, ok := authDataMap[tenantIDKey]; ok {
		dataMap[tenantIDKey] = tenantID
	}
	if domainID, ok := authDataMap[domainIDKey]; ok {
		dataMap[domainIDKey] = domainID
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

	if !applyFilterToResource(resource.Data(), getFilterFromPolicy(auth, resource.ID(), resourceSchema, policy, schema.ActionCreate)) {
		return ResourceError{err, "", Unauthorized}
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

func buildAuthDataMap(resourceSchema *schema.Schema, auth schema.Authorization,
	dataMap map[string]interface{}) (map[string]interface{}, error) {

	authMap := map[string]interface{}{}

	tenantID, provided := dataMap[tenantIDKey]
	expected := resourceSchema.HasPropertyID(tenantIDKey)
	if provided || expected {
		if !provided {
			tenantID = auth.TenantID()
			if tenantID == "" {
				err := errors.New("A non-empty tenant_id should be provided in the request")
				return nil, err
			}
		}

		authMap[tenantIDKey] = tenantID
		authMap["tenant_name"] = auth.TenantName()
	}

	domainID, provided := dataMap[domainIDKey]
	expected = resourceSchema.HasPropertyID(domainIDKey)
	if provided || expected {
		if !provided {
			domainID = auth.DomainID()
		}

		authMap[domainIDKey] = domainID
		authMap["domain_name"] = auth.DomainName()
	}

	return authMap, nil
}

//CreateResourceInTransaction create db resource model in transaction
func CreateResourceInTransaction(context middleware.Context, resourceSchema *schema.Schema, resource *schema.Resource) error {
	defer MeasureRequestTime(time.Now(), "create.in_tx", resource.Schema().ID)
	manager := schema.GetManager()
	mainTransaction := mustGetTransaction(context)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	if err := ValidateAttachmentsForResource(context, resourceSchema, resource.Data()); err != nil {
		return err
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
	}
	if _, err := mainTransaction.Create(mustGetContext(context), resource); err != nil {
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

	fillDataMapWithTenantAndDomainNames(auth, dataMap)

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
			exists, err := checkIfResourceExistsForPolicy(mustGetContext(context), auth, resourceID, resourceSchema,
				policy, schema.ActionUpdate, mustGetTransaction(context))
			if err != nil {
				return err
			}
			if !exists {
				return ResourceError{transaction.ErrResourceNotFound, "", Unauthorized}
			}
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
	mainTransaction := mustGetTransaction(context)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	filter := transaction.IDFilter(resourceID)
	if _, err := resourceSchema.GetPropertyByID(tenantIDKey); err == nil && tenantIDs != nil {
		filter[tenantIDKey] = tenantIDs
	}
	if _, err := resourceSchema.GetPropertyByID(domainIDKey); err == nil && domainIDs != nil {
		filter[domainIDKey] = domainIDs
	}

	var resource *schema.Resource
	var err error
	switch resourceSchema.GetLockingPolicy("update") {
	case schema.NoLocking:
		resource, err = mainTransaction.Fetch(mustGetContext(context), resourceSchema, filter, nil)
	case schema.LockRelatedResources:
		resource, err = mainTransaction.LockFetch(mustGetContext(context), resourceSchema,
			filter, schema.LockRelatedResources, nil)
	case schema.SkipRelatedResources:
		resource, err = mainTransaction.LockFetch(mustGetContext(context), resourceSchema,
			filter, schema.SkipRelatedResources, nil)
	}

	if err != nil {
		return ResourceError{err, err.Error(), WrongQuery}
	}

	if err := validate(context, &dataMap, resourceSchema.ValidateOnUpdate); err != nil {
		return err
	}

	if err := ValidateAttachmentsForResource(context, resourceSchema, dataMap); err != nil {
		return err
	}

	policy := context["policy"].(*schema.Policy)
	// apply property filter
	currCond := policy.GetCurrentResourceCondition()
	err = currCond.ApplyPropertyConditionFilter(schema.ActionUpdate, resource.Data(), dataMap)
	if err != nil {
		return ResourceError{err, "", Unauthorized}
	}

	data := resource.CloneWithUpdate(dataMap)
	context["resource"] = data

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

func fetchResource(resourceID string, resourceSchema *schema.Schema,
	tx transaction.Transaction, context middleware.Context) (*schema.Resource, error) {
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
	mainTransaction := mustGetTransaction(context)
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
func ActionResource(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema,
	action schema.Action, resourceID string, data map[string]interface{},
) error {
	defer MeasureRequestTime(time.Now(), action.ID, resourceSchema.ID)

	if err := checkIfActionIsAllowedForUser(context, dataStore, data, resourceSchema, action, resourceID); err != nil {
		return err
	}

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

	_, extensionReturnedResponse := context["response"]

	if extensionReturnedResponse || action.HasResponseOwnership {
		return nil
	}

	return fmt.Errorf("no response")
}

func checkIfActionIsAllowedForUser(context middleware.Context, dataStore db.DB, dataMap map[string]interface{},
	resourceSchema *schema.Schema, action schema.Action, resourceID string) error {
	auth := context["auth"].(schema.Authorization)
	policyPath, isPlural := getResourcePathFromCustomAction(action, resourceSchema, resourceID)
	policy, err := LoadPolicy(context, action.ID, policyPath, auth)

	if err != nil {
		return err
	}

	fillDataMapWithTenantAndDomainNames(auth, dataMap)

	err = policy.Check(action.ID, auth, dataMap)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}

	currCond := policy.GetCurrentResourceCondition()
	err = currCond.ApplyPropertyConditionFilter(action.ID, dataMap, nil)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}

	if isPlural {
		return nil
	}

	if err := resourceTransactionWithContext(context, dataStore, transaction.GetIsolationLevel(resourceSchema, action.ID),
		func() error {
			exists, err := checkIfResourceExistsForPolicy(mustGetContext(context), auth, resourceID,
				resourceSchema, policy, action.ID, mustGetTransaction(context))
			if err != nil {
				return err
			}
			if !exists {
				return ResourceError{transaction.ErrResourceNotFound, "", NotFound}
			}
			return nil
		},
	); err != nil {
		return err
	}
	return nil
}

func fillDataMapWithTenantAndDomainNames(auth schema.Authorization, dataMap map[string]interface{}) {
	if tenantID, ok := dataMap[tenantIDKey]; ok && tenantID != nil {
		dataMap["tenant_name"] = auth.TenantName()
	}
	if domainID, ok := dataMap[domainIDKey]; ok && domainID != nil {
		dataMap["domain_name"] = auth.DomainName()
	}
}

func getResourcePathFromCustomAction(action schema.Action, resourceSchema *schema.Schema, resourceID string) (string, bool) {
	if strings.Contains(action.Path, ":id") {
		return strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1), false
	}
	return resourceSchema.GetPluralURL(), true
}

func validateAttachment(
	context middleware.Context,
	policy *schema.Policy,
	auth schema.Authorization,
	resourceSchema *schema.Schema,
	dataMap map[string]interface{},
) error {
	log.Debug("Checking tenant isolation policy: %s", policy.ID)
	relationPropertyName := policy.GetRelationPropertyName()
	if relationPropertyName == "*" {
		for path := range resourceSchema.GetAllPropertiesFullyQualifiedMap() {
			err := validateAttachmentRelationUnderPath(context, policy, auth, dataMap, resourceSchema, path)
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		return validateAttachmentRelationUnderPath(context, policy, auth, dataMap, resourceSchema, relationPropertyName)
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

func validateAttachmentRelationUnderPath(
	context middleware.Context,
	policy *schema.Policy,
	auth schema.Authorization,
	data map[string]interface{},
	resourceSchema *schema.Schema,
	path string,
) error {
	traversal := newDataTraversal(resourceSchema, data)

	log.Debug("Checking tenant isolation for property %s in schema %s", path, resourceSchema.ID)

	for _, key := range strings.Split(path, ".") {
		if key == schema.ItemPropertyID {
			if err := traversal.advanceByItem(); err != nil {
				return err
			}
		} else {
			if err := traversal.advanceByKey(key); err != nil {
				return err
			}
		}
	}

	currProp := traversal.getCurrentProperty()

	if currProp == nil {
		return fmt.Errorf("Property under path %s not found in schema %s", path, resourceSchema.ID)
	}

	if currProp.Relation == "" {
		// Not a relation, so skip
		log.Debug("The property %s is not a relation, skipping tenant isolation check", path)
		return nil
	}

	dataSet := traversal.getCurrentDataSet()
	log.Debug("Will check %d resource IDs", len(dataSet))
	for _, dataPiece := range dataSet {
		relatedResourceID, _ := dataPiece.(string)
		log.Debug("Checking ID %s", relatedResourceID)
		err := validateAttachmentRelation(context, policy, auth, currProp, relatedResourceID, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateAttachmentRelation(
	context middleware.Context,
	policy *schema.Policy,
	auth schema.Authorization,
	property *schema.Property,
	relatedResourceID string,
	data map[string]interface{},
) error {
	if relatedResourceID == "" {
		log.Debug("Skipping tenant isolation check because of empty ID")
		return nil
	}

	manager := schema.GetManager()
	relatedSchema, _ := manager.Schema(property.Relation)

	otherCond := policy.GetOtherResourceCondition()
	filter := transaction.IDFilter(relatedResourceID)
	extendFilterByTenantAndDomain(relatedSchema, filter, schema.ActionRead, otherCond, auth)
	otherCond.AddCustomFilters(relatedSchema, filter, auth)

	options := &transaction.ViewOptions{Details: true}

	mainTransaction := mustGetTransaction(context)
	relatedRes, err := mainTransaction.Fetch(mustGetContext(context), relatedSchema, filter, options)

	if err != nil {
		if policy.IsDeny() {
			return nil
		} else {
			log.Debug(
				"Tenant isolation failed: ID %s is not visible for tenant %s in domain %s",
				relatedResourceID, auth.TenantID(), auth.DomainID(),
			)
			return errorBadRequest(relatedSchema.ID, relatedResourceID)
		}
	}

	if !checkRelatedResource(relatedRes, data, policy) {
		log.Debug(
			"Tenant isolation failed: tenat or domain mismatch %s (tenant: %s, domain: %s)",
			relatedResourceID, auth.TenantID(), auth.DomainID(),
		)
		return errorBadRequest(relatedSchema.ID, relatedResourceID)
	}

	err = otherCond.ApplyPropertyConditionFilter(schema.ActionAttach, relatedRes.Data(), nil)
	if policy.IsDeny() {
		if err == nil {
			log.Debug(
				"Tenant isolation failed: property condition filter rejected ID %s (tenant: %s, domain: %s) by policy %s",
				relatedResourceID, auth.TenantID(), auth.DomainID(), policy.ID,
			)
			return errorBadRequest(relatedSchema.ID, relatedResourceID)
		}
	} else if err != nil {
		log.Debug(
			"Tenant isolation failed: property condition filter rejected ID %s (tenant: %s, domain: %s)",
			relatedResourceID, auth.TenantID(), auth.DomainID(),
		)
		return errorNotFound(relatedSchema.ID, relatedResourceID)
	}
	return nil
}

func checkRelatedResource(relatedResource *schema.Resource, data map[string]interface{}, policy *schema.Policy) bool {
	rc := policy.GetOtherResourceCondition()
	if rc.SkipTenantDomainCheck() {
		return true
	}
	return equalStringKeys(data, relatedResource.Data(), tenantIDKey) &&
		equalStringKeys(data, relatedResource.Data(), domainIDKey)
}

func equalStringKeys(first, second map[string]interface{}, key string) bool {
	if firstValue, ok := first[key].(string); ok {
		if secondValue, ok := second[key].(string); ok {
			return firstValue == secondValue
		}
	}
	return true
}

func extendFilterByTenantAndDomain(
	schema *schema.Schema,
	filter transaction.Filter, action string,
	rc *schema.ResourceCondition, auth schema.Authorization,
) {
	tenantFilter, domainFilter := rc.GetTenantAndDomainFilters(action, auth)
	if _, err := schema.GetPropertyByID(tenantIDKey); err == nil && len(tenantFilter) > 0 {
		filter[tenantIDKey] = tenantFilter
	}
	if _, err := schema.GetPropertyByID(domainIDKey); err == nil && len(domainFilter) > 0 {
		filter[domainIDKey] = domainFilter
	}
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

type dataTraversal struct {
	currentProperty *schema.Property
	currentChildren []schema.Property
	dataSet         []interface{}
}

func newDataTraversal(schema *schema.Schema, data interface{}) *dataTraversal {
	return &dataTraversal{
		currentProperty: nil,
		currentChildren: schema.Properties,
		dataSet:         []interface{}{data},
	}
}

func (traversal *dataTraversal) advanceByItem() error {
	if traversal.currentProperty == nil || traversal.currentProperty.Items == nil {
		return errors.New("cannot access array: currently situated on non-array property")
	}

	var newDataSet []interface{}
	for _, dataPiece := range traversal.dataSet {
		if dataPiece == nil {
			continue
		}
		switch dataPieceArray := dataPiece.(type) {
		case []interface{}:
			newDataSet = append(newDataSet, dataPieceArray...)
		case []map[string]interface{}:
			for _, element := range dataPieceArray {
				newDataSet = append(newDataSet, element)
			}
		default:
			return errors.Errorf("expected []interface{}, []map[string]interface{} or nil while advancing by item, got: %#v", dataPiece)
		}
		dataPieceArray, _ := dataPiece.([]interface{})
		newDataSet = append(newDataSet, dataPieceArray...)
	}

	traversal.currentProperty = traversal.currentProperty.Items
	traversal.currentChildren = traversal.currentProperty.Properties
	traversal.dataSet = newDataSet
	return nil
}

func (traversal *dataTraversal) advanceByKey(key string) error {
	var child *schema.Property
	for i := range traversal.currentChildren {
		if traversal.currentChildren[i].ID == key {
			child = &traversal.currentChildren[i]
			break
		}
	}

	if child == nil {
		return fmt.Errorf("property of name \"%s\" not found", key)
	}

	var newDataSet []interface{}
	for _, dataPiece := range traversal.dataSet {
		if dataPiece == nil {
			continue
		}
		dataPieceMap, ok := dataPiece.(map[string]interface{})
		if !ok {
			return errors.Errorf("expected map[string]interface{} or nil while advancing by key \"%s\", got: %#v", key, dataPiece)
		}
		if dataPieceUnderKey, _ := dataPieceMap[key]; dataPieceUnderKey != nil {
			newDataSet = append(newDataSet, dataPieceUnderKey)
		}
	}

	traversal.currentProperty = child
	traversal.currentChildren = child.Properties
	traversal.dataSet = newDataSet
	return nil
}

func (traversal *dataTraversal) getCurrentProperty() *schema.Property {
	return traversal.currentProperty
}

func (traversal *dataTraversal) getCurrentDataSet() []interface{} {
	return traversal.dataSet
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

func mustGetTransaction(requestContext middleware.Context) transaction.Transaction {
	return requestContext["transaction"].(transaction.Transaction)
}

func traceIdOrEmpty(requestContext middleware.Context) string {
	if traceId, hasTraceId := requestContext["trace_id"]; hasTraceId {
		return traceId.(string)
	}

	return ""
}
