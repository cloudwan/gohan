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
	"fmt"
	"strings"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/pagination"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/extension"

	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
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
	hlsearch

	Unauthorized
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

//InTransaction executes function in the db transaction and set it to the context
func InTransaction(context middleware.Context, dataStore db.DB, level transaction.Type, f func() error) error {
	if context["transaction"] != nil {
		return fmt.Errorf("cannot create nested transaction")
	}
	aTransaction, err := dataStore.Begin()
	if err != nil {
		return fmt.Errorf("cannot create transaction: %v", err)
	}
	defer aTransaction.Close()
	err = aTransaction.SetIsolationLevel(level)
	if err != nil {
		return fmt.Errorf("error when setting isolation level '%s': %s", level, err)
	}
	context["transaction"] = aTransaction

	err = f()
	if err != nil {
		return err
	}

	err = aTransaction.Commit()
	if err != nil {
		return fmt.Errorf("commit error : %s", err)
	}
	delete(context, "transaction")
	return nil
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
	for _, resource := range resources {
		resourceMap := resource.(map[string]interface{})
		if err := policy.ApplyPropertyConditionFilter(schema.ActionRead, resourceMap, nil); err != nil {
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
	if err := policy.ApplyPropertyConditionFilter(schema.ActionRead, resourceMap, nil); err != nil {
		return err
	}
	response[resourceSchema.Singular] = policy.RemoveHiddenProperty(resourceMap)

	return nil
}

//GetResources returns specified resources without calling non in_transaction events
func GetResources(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema, filter map[string]interface{}, paginator *pagination.Paginator) error {
	return InTransaction(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionRead),
		func() error {
			return GetResourcesInTransaction(context, resourceSchema, filter, paginator)
		},
	)
}

//GetResourcesInTransaction returns specified resources without calling non in_transaction events
func GetResourcesInTransaction(context middleware.Context, resourceSchema *schema.Schema, filter map[string]interface{}, paginator *pagination.Paginator) error {
	mainTransaction := context["transaction"].(transaction.Transaction)
	response := map[string]interface{}{}

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("no environment for schema")
	}

	if err := extension.HandleEvent(context, environment, "pre_list_in_transaction"); err != nil {
		return err
	}

	list, total, err := mainTransaction.List(resourceSchema, filter, paginator)
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

	if err := extension.HandleEvent(context, environment, "post_list_in_transaction"); err != nil {
		return err
	}
	return nil
}

//FilterFromQueryParameter makes list filter from query
func FilterFromQueryParameter(resourceSchema *schema.Schema,
	queryParameters map[string][]string) map[string]interface{} {
	filter := map[string]interface{}{}
	for key, value := range queryParameters {
		if _, err := resourceSchema.GetPropertyByID(key); err != nil {
			log.Info("Resource %s does not have %s property, ignoring filter.", resourceSchema, key)
			continue
		}
		filter[key] = value
	}
	return filter
}

// GetMultipleResources returns all resources specified by the schema and query parameters
func GetMultipleResources(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema, queryParameters map[string][]string) error {
	log.Debug("Start get multiple resources!!")
	auth := context["auth"].(schema.Authorization)
	policy, err := loadPolicy(context, "read", resourceSchema.GetPluralURL(), auth)
	if err != nil {
		return err
	}

	filter := FilterFromQueryParameter(resourceSchema, queryParameters)

	if policy.RequireOwner() {
		filter["tenant_id"] = policy.GetTenantIDFilter(schema.ActionRead, auth.TenantID())
	}
	filter = policy.RemoveHiddenProperty(filter)

	paginator, err := pagination.FromURLQuery(resourceSchema, queryParameters)
	if err != nil {
		return ResourceError{err, err.Error(), WrongQuery}
	}
	context["policy"] = policy

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	if err := extension.HandleEvent(context, environment, "pre_list"); err != nil {
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

	if err := extension.HandleEvent(context, environment, "post_list"); err != nil {
		return err
	}

	if err := ApplyPolicyForResources(context, resourceSchema); err != nil {
		return err
	}

	return nil
}

// GetSingleResource returns the resource specified by the schema and ID
func GetSingleResource(context middleware.Context, dataStore db.DB, resourceSchema *schema.Schema, resourceID string) error {
	context["id"] = resourceID
	auth := context["auth"].(schema.Authorization)
	policy, err := loadPolicy(context, "read", strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1), auth)
	if err != nil {
		return err
	}
	context["policy"] = policy

	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	if err := extension.HandleEvent(context, environment, "pre_show"); err != nil {
		return err
	}
	if rawResponse, ok := context["response"]; ok {
		if _, ok := rawResponse.(map[string]interface{}); ok {
			return nil
		}
		return fmt.Errorf("extension returned invalid JSON: %v", rawResponse)
	}

	if err := InTransaction(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionRead),
		func() error {
			return GetSingleResourceInTransaction(context, resourceSchema, resourceID, policy.GetTenantIDFilter(schema.ActionRead, auth.TenantID()))
		},
	); err != nil {
		return err
	}

	if err := extension.HandleEvent(context, environment, "post_show"); err != nil {
		return err
	}
	if err := ApplyPolicyForResource(context, resourceSchema); err != nil {
		return ResourceError{err, "", NotFound}
	}
	return nil
}

//GetSingleResourceInTransaction get resource in single transaction
func GetSingleResourceInTransaction(context middleware.Context, resourceSchema *schema.Schema, resourceID string, tenantIDs []string) (err error) {
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("no environment for schema")
	}

	if err := extension.HandleEvent(context, environment, "pre_show_in_transaction"); err != nil {
		return err
	}
	if rawResponse, ok := context["response"]; ok {
		if _, ok := rawResponse.(map[string]interface{}); ok {
			return nil
		}
		return fmt.Errorf("extension returned invalid JSON: %v", rawResponse)
	}
	filter := transaction.IDFilter(resourceID)
	if tenantIDs != nil {
		filter["tenant_id"] = tenantIDs
	}
	object, err := mainTransaction.Fetch(resourceSchema, filter)

	if err != nil || object == nil {
		return ResourceError{err, "", NotFound}
	}

	response := map[string]interface{}{}
	response[resourceSchema.Singular] = object.Data()
	context["response"] = response

	if err := extension.HandleEvent(context, environment, "post_show_in_transaction"); err != nil {
		return err
	}
	return
}

// CreateOrUpdateResource updates resource if it existed and otherwise creates it and returns true.
func CreateOrUpdateResource(
	context middleware.Context,
	dataStore db.DB, identityService middleware.IdentityService,
	resourceSchema *schema.Schema,
	resourceID string, dataMap map[string]interface{},
) (bool, error) {
	auth := context["auth"].(schema.Authorization)

	//LoadPolicy
	policy, err := loadPolicy(context, "update", strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1), auth)
	if err != nil {
		return false, err
	}

	preTransaction, err := dataStore.Begin()
	if err != nil {
		return false, fmt.Errorf("cannot create transaction: %v", err)
	}
	tenantIDs := policy.GetTenantIDFilter(schema.ActionUpdate, auth.TenantID())
	filter := transaction.IDFilter(resourceID)
	if tenantIDs != nil {
		filter["tenant_id"] = tenantIDs
	}
	_, fetchErr := preTransaction.Fetch(resourceSchema, filter)
	preTransaction.Close()

	if fetchErr != nil {
		dataMap["id"] = resourceID
		if err := CreateResource(context, dataStore, identityService, resourceSchema, dataMap); err != nil {
			return false, err
		}
		return true, err
	}

	return false, UpdateResource(context, dataStore, identityService, resourceSchema, resourceID, dataMap)
}

// CreateResource creates the resource specified by the schema and dataMap
func CreateResource(
	context middleware.Context,
	dataStore db.DB,
	identityService middleware.IdentityService,
	resourceSchema *schema.Schema,
	dataMap map[string]interface{},
) error {
	manager := schema.GetManager()
	// Load environment
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)

	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	auth := context["auth"].(schema.Authorization)

	//LoadPolicy
	policy, err := loadPolicy(context, "create", resourceSchema.GetPluralURL(), auth)
	if err != nil {
		return err
	}

	_, err = resourceSchema.GetPropertyByID("tenant_id")
	if _, ok := dataMap["tenant_id"]; err == nil && !ok {
		dataMap["tenant_id"] = context["tenant_id"]
	}

	if tenantID, ok := dataMap["tenant_id"]; ok && tenantID != nil {
		dataMap["tenant_name"], err = identityService.GetTenantName(tenantID.(string))
		if err != nil {
			return ResourceError{err, err.Error(), Unauthorized}
		}
	}

	//Apply policy for api input
	err = policy.Check(schema.ActionCreate, auth, dataMap)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}
	delete(dataMap, "tenant_name")

	// apply property filter
	err = policy.ApplyPropertyConditionFilter(schema.ActionCreate, dataMap, nil)
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}
	context["resource"] = dataMap
	if id, ok := dataMap["id"]; !ok || id == "" {
		dataMap["id"] = uuid.NewV4().String()
	}
	context["id"] = dataMap["id"]

	if err := extension.HandleEvent(context, environment, "pre_create"); err != nil {
		return err
	}

	if resourceData, ok := context["resource"].(map[string]interface{}); ok {
		dataMap = resourceData
	}

	//Validation
	err = resourceSchema.ValidateOnCreate(dataMap)
	if err != nil {
		return ResourceError{err, fmt.Sprintf("Validation error: %s", err), WrongData}
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

	if err := InTransaction(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionCreate),
		func() error {
			return CreateResourceInTransaction(context, resource)
		},
	); err != nil {
		return err
	}

	if err := extension.HandleEvent(context, environment, "post_create"); err != nil {
		return err
	}

	if err := ApplyPolicyForResource(context, resourceSchema); err != nil {
		return ResourceError{err, "", Unauthorized}
	}
	return nil
}

//CreateResourceInTransaction craete db resource model in transaction
func CreateResourceInTransaction(context middleware.Context, resource *schema.Resource) error {
	resourceSchema := resource.Schema()
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	if err := extension.HandleEvent(context, environment, "pre_create_in_transaction"); err != nil {
		return err
	}
	if err := mainTransaction.Create(resource); err != nil {
		log.Debug("%s transaction error", err)
		return ResourceError{
			err,
			fmt.Sprintf("Failed to store data in database: %v", err),
			CreateFailed}
	}

	response := map[string]interface{}{}
	response[resourceSchema.Singular] = resource.Data()
	context["response"] = response

	if err := extension.HandleEvent(context, environment, "post_create_in_transaction"); err != nil {
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

	context["id"] = resourceID

	//load environment
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	auth := context["auth"].(schema.Authorization)

	//load policy
	policy, err := loadPolicy(context, "update", strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1), auth)
	if err != nil {
		return err
	}
	context["policy"] = policy

	//fillup default values
	if tenantID, ok := dataMap["tenant_id"]; ok && tenantID != nil {
		dataMap["tenant_name"], err = identityService.GetTenantName(tenantID.(string))
	}
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}

	//check policy
	err = policy.Check(schema.ActionUpdate, auth, dataMap)
	delete(dataMap, "tenant_name")
	if err != nil {
		return ResourceError{err, err.Error(), Unauthorized}
	}
	context["resource"] = dataMap

	if err := extension.HandleEvent(context, environment, "pre_update"); err != nil {
		return err
	}

	if resourceData, ok := context["resource"].(map[string]interface{}); ok {
		dataMap = resourceData
	}

	if err := InTransaction(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionUpdate),
		func() error {
			return UpdateResourceInTransaction(context, resourceSchema, resourceID, dataMap, policy.GetTenantIDFilter(schema.ActionUpdate, auth.TenantID()))
		},
	); err != nil {
		return err
	}

	if err := extension.HandleEvent(context, environment, "post_update"); err != nil {
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
	dataMap map[string]interface{}, tenantIDs []string) error {

	manager := schema.GetManager()
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	filter := transaction.IDFilter(resourceID)
	if tenantIDs != nil {
		filter["tenant_id"] = tenantIDs
	}
	resource, err := mainTransaction.Fetch(
		resourceSchema, filter)
	if err != nil {
		return ResourceError{err, err.Error(), WrongQuery}
	}

	policy := context["policy"].(*schema.Policy)
	// apply property filter
	err = policy.ApplyPropertyConditionFilter(schema.ActionUpdate, resource.Data(), dataMap)
	if err != nil {
		return ResourceError{err, "", Unauthorized}
	}

	err = resource.Update(dataMap)
	if err != nil {
		return ResourceError{err, err.Error(), WrongData}
	}
	context["resource"] = resource.Data()

	if err := extension.HandleEvent(context, environment, "pre_update_in_transaction"); err != nil {
		return err
	}

	dataMap, ok = context["resource"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("Resource not JSON: %s", err)
	}
	resource, err = manager.LoadResource(resourceSchema.ID, dataMap)
	if err != nil {
		return fmt.Errorf("Loading Resource failed: %s", err)
	}

	err = mainTransaction.Update(resource)
	if err != nil {
		return ResourceError{err, fmt.Sprintf("Failed to store data in database: %v", err), UpdateFailed}
	}

	response := map[string]interface{}{}
	response[resourceSchema.Singular] = resource.Data()
	context["response"] = response

	if err := extension.HandleEvent(context, environment, "post_update_in_transaction"); err != nil {
		return err
	}

	return nil
}

// DeleteResource deletes the resource specified by the schema and ID
func DeleteResource(context middleware.Context,
	dataStore db.DB,
	resourceSchema *schema.Schema,
	resourceID string,
) error {
	context["id"] = resourceID
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}
	auth := context["auth"].(schema.Authorization)
	policy, err := loadPolicy(context, "delete", strings.Replace(resourceSchema.GetSingleURL(), ":id", resourceID, 1), auth)
	if err != nil {
		return err
	}
	context["policy"] = policy
	preTransaction, err := dataStore.Begin()
	if err != nil {
		return fmt.Errorf("cannot create transaction: %v", err)
	}
	tenantIDs := policy.GetTenantIDFilter(schema.ActionDelete, auth.TenantID())
	filter := transaction.IDFilter(resourceID)
	if tenantIDs != nil {
		filter["tenant_id"] = tenantIDs
	}
	resource, fetchErr := preTransaction.Fetch(resourceSchema, filter)
	preTransaction.Close()

	if resource != nil {
		context["resource"] = resource.Data()
	}

	if err := extension.HandleEvent(context, environment, "pre_delete"); err != nil {
		return err
	}
	if fetchErr != nil {
		return ResourceError{err, "", NotFound}
	}
	if err := InTransaction(
		context, dataStore,
		transaction.GetIsolationLevel(resourceSchema, schema.ActionDelete),
		func() error {
			return DeleteResourceInTransaction(context, resourceSchema, resourceID)
		},
	); err != nil {
		return err
	}
	if err := extension.HandleEvent(context, environment, "post_delete"); err != nil {
		return err
	}
	return nil
}

//DeleteResourceInTransaction deletes resources in a transaction
func DeleteResourceInTransaction(context middleware.Context, resourceSchema *schema.Schema, resourceID string) error {
	mainTransaction := context["transaction"].(transaction.Transaction)
	environmentManager := extension.GetManager()
	environment, ok := environmentManager.GetEnvironment(resourceSchema.ID)
	if !ok {
		return fmt.Errorf("No environment for schema")
	}

	auth := context["auth"].(schema.Authorization)
	policy := context["policy"].(*schema.Policy)
	tenantIDs := policy.GetTenantIDFilter(schema.ActionDelete, auth.TenantID())
	filter := transaction.IDFilter(resourceID)
	if tenantIDs != nil {
		filter["tenant_id"] = tenantIDs
	}
	resource, err := mainTransaction.Fetch(resourceSchema, filter)
	log.Debug("%s %s", resource, err)
	if err != nil {
		return err
	}
	if resource != nil {
		context["resource"] = resource.Data()
	}
	// apply property filter
	err = policy.ApplyPropertyConditionFilter(schema.ActionUpdate, resource.Data(), nil)
	if err != nil {
		return ResourceError{err, "", Unauthorized}
	}
	if err := extension.HandleEvent(context, environment, "pre_delete_in_transaction"); err != nil {
		return err
	}

	err = mainTransaction.Delete(resourceSchema, resourceID)
	if err != nil {
		return ResourceError{err, "", DeleteFailed}
	}

	if err := extension.HandleEvent(context, environment, "post_delete_in_transaction"); err != nil {
		return err
	}
	return nil
}

// ActionResource runs custom action on resource
func ActionResource(context middleware.Context, dataStore db.DB, identityService middleware.IdentityService,
	resourceSchema *schema.Schema, action schema.Action, resourceID string, data interface{},
) error {
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

	err := extension.HandleEvent(context, environment, action.ID)
	if err != nil {
		return err
	}

	if _, ok := context["response"]; ok {
		return nil
	}

	return fmt.Errorf("no response")
}

func loadPolicy(context middleware.Context, action, path string, auth schema.Authorization) (*schema.Policy, error) {
	manager := schema.GetManager()
	policy, role := manager.PolicyValidate(action, path, auth)
	if policy == nil {
		err := fmt.Errorf(fmt.Sprintf("No matching policy: %s %s", action, path))
		return nil, ResourceError{err, err.Error(), Unauthorized}
	}
	context["policy"] = policy
	context["role"] = role
	return policy, nil
}
