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

package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/job"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/server/middleware"
	"github.com/cloudwan/gohan/server/resources"
	"github.com/cloudwan/gohan/sync"
	"github.com/drone/routes"
	"github.com/go-martini/martini"
)

var (
	exceptionObjectDoestNotContainKeyError  = "Exception obejct does not contain '%s'"
	exceptionPropertyIsNotExpectedTypeError = "Exception property '%s' is not '%s'"
)

func authorization(w http.ResponseWriter, r *http.Request, action, path string, s *schema.Schema, auth schema.Authorization) (*schema.Policy, *schema.Role) {
	manager := schema.GetManager()
	log.Debug("[authorization*] %s %v", action, auth)
	if auth == nil {
		return schema.NewEmptyPolicy(), nil
	}
	policy, role := manager.PolicyValidate(action, path, auth)
	if policy == nil {
		log.Debug("No policy match: %s %s", action, path)
		return nil, nil
	}
	return policy, role
}

func addParamToQuery(r *http.Request, key, value string) {
	r.URL.RawQuery += "&" + key + "=" + value
}

func addJSONContentTypeHeader(w http.ResponseWriter) {
	w.Header().Add("Content-Type", "application/json")
}

func removeResourceWrapper(s *schema.Schema, dataMap map[string]interface{}) map[string]interface{} {
	if innerData, ok := dataMap[s.Singular]; ok {
		if innerDataMap, ok := innerData.(map[string]interface{}); ok {
			return innerDataMap
		}
	}

	return dataMap
}

func problemToResponseCode(problem resources.ResourceProblem) int {
	switch problem {
	case resources.InternalServerError:
		return http.StatusInternalServerError
	case resources.WrongQuery, resources.WrongData:
		return http.StatusBadRequest
	case resources.NotFound:
		return http.StatusNotFound
	case resources.DeleteFailed, resources.CreateFailed, resources.UpdateFailed:
		return http.StatusConflict
	case resources.Unauthorized:
		return http.StatusUnauthorized
	}
	return http.StatusInternalServerError
}

func unwrapExtensionException(exceptionInfo map[string]interface{}) (message map[string]interface{}, code int) {
	messageRaw, ok := exceptionInfo["message"]
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf(exceptionObjectDoestNotContainKeyError, "message")}, http.StatusInternalServerError
	}
	nameRaw, ok := exceptionInfo["name"]
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf(exceptionObjectDoestNotContainKeyError, "name")}, http.StatusInternalServerError
	}
	name, ok := nameRaw.(string)
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf(exceptionPropertyIsNotExpectedTypeError, "name", "string")}, http.StatusInternalServerError
	}
	code = 400
	var err error
	switch name {
	case "CustomException":
		codeRaw, ok := exceptionInfo["code"]
		if !ok {
			return map[string]interface{}{"error": fmt.Sprintf(exceptionObjectDoestNotContainKeyError, "code")}, http.StatusInternalServerError
		}
		code, err = strconv.Atoi(fmt.Sprint(codeRaw))
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf(exceptionPropertyIsNotExpectedTypeError, "code", "int")}, http.StatusInternalServerError
		}
	case "ResourceException":
		problemRaw, ok := exceptionInfo["problem"]
		if !ok {
			return map[string]interface{}{"error": fmt.Sprintf(exceptionObjectDoestNotContainKeyError, "problem")}, http.StatusInternalServerError
		}
		problem, err := strconv.Atoi(fmt.Sprint(problemRaw))
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf(exceptionPropertyIsNotExpectedTypeError, "problem", "int")}, http.StatusInternalServerError
		}
		code = problemToResponseCode(resources.ResourceProblem(problem))
	case "ExtensionException":
		innerExceptionInfoRaw, ok := exceptionInfo["inner_exception"]
		if !ok {
			return map[string]interface{}{"error": fmt.Sprintf(exceptionObjectDoestNotContainKeyError, "inner_exception")}, http.StatusInternalServerError
		}
		innerExceptionInfo, ok := innerExceptionInfoRaw.(map[string]interface{})
		if !ok {
			return map[string]interface{}{"error": fmt.Sprintf(exceptionPropertyIsNotExpectedTypeError, "inner_exception", "map[string]interface{}")}, http.StatusInternalServerError
		}
		_, code = unwrapExtensionException(innerExceptionInfo)
	}
	if 200 <= code && code < 300 {
		message, ok = messageRaw.(map[string]interface{})
		if !ok {
			return map[string]interface{}{"error": fmt.Sprintf(exceptionPropertyIsNotExpectedTypeError, "message", "map[string]interface{}")}, http.StatusInternalServerError
		}
	} else {
		message = map[string]interface{}{"error": fmt.Sprintf("%v", messageRaw)}
	}
	return message, code
}

func handleError(writer http.ResponseWriter, err error) {
	switch err := err.(type) {
	default:
		middleware.HTTPJSONError(writer, err.Error(), http.StatusInternalServerError)
	case resources.ResourceError:
		code := problemToResponseCode(err.Problem)
		middleware.HTTPJSONError(writer, err.Message, code)
	case extension.Error:
		message, code := unwrapExtensionException(err.ExceptionInfo)
		if 200 <= code && code < 300 {
			writer.WriteHeader(code)
			routes.ServeJson(writer, message)
		} else {
			middleware.HTTPJSONError(writer, message["error"].(string), code)
		}
	}
}

func fillInContext(context middleware.Context, db db.DB,
	r *http.Request, w http.ResponseWriter,
	s *schema.Schema, p martini.Params, sync sync.Sync,
	identityService middleware.IdentityService,
	queue *job.Queue) {
	context["path"] = r.URL.Path
	context["http_request"] = r
	context["http_response"] = w
	context["schema"] = s
	params := map[string]interface{}{}
	for key, value := range p {
		params[key] = value
	}
	context["params"] = params
	context["sync"] = sync
	context["db"] = db
	context["queue"] = queue
	context["identity_service"] = identityService
	context["service_auth"], _ = identityService.GetServiceAuthorization()
	context["openstack_client"] = identityService.GetClient()
}

//MapRouteBySchema setup api route by schema
func MapRouteBySchema(server *Server, dataStore db.DB, s *schema.Schema) {
	if s.IsAbstract() {
		return
	}
	route := server.martini

	singleURL := s.GetSingleURL()
	pluralURL := s.GetPluralURL()
	singleURLWithParents := s.GetSingleURLWithParents()
	pluralURLWithParents := s.GetPluralURLWithParents()

	//load extension environments
	environmentManager := extension.GetManager()
	if _, ok := environmentManager.GetEnvironment(s.ID); !ok {
		env, err := server.NewEnvironmentForPath(s.ID, pluralURL)
		if err != nil {
			log.Fatal(fmt.Sprintf("[%s] %v", pluralURL, err))
		}
		environmentManager.RegisterEnvironment(s.ID, env)
	}

	log.Debug("[Plural Path] %s", pluralURL)
	log.Debug("[Singular Path] %s", singleURL)
	log.Debug("[Plural Path With Parents] %s", pluralURLWithParents)
	log.Debug("[Singular Path With Parents] %s", singleURLWithParents)

	//setup list route
	getPluralFunc := func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addJSONContentTypeHeader(w)
		fillInContext(context, dataStore, r, w, s, p, server.sync, identityService, server.queue)

		var newEtag string
		var err error
		if r.Header.Get(LongPollHeader) != "" {
			oldEtag := r.Header.Get(LongPollHeader)
			getResource := func(context middleware.Context) error {
				return resources.GetMultipleResources(context, dataStore, s, r.URL.Query())
			}
			newEtag, err = server.longPoll.GetOrWait(r.URL.Path, oldEtag, context, getResource, calculateResponseEtag)
		} else {
			err = resources.GetMultipleResources(context, dataStore, s, r.URL.Query())
			if err == nil {
				newEtag = calculateResponseEtag(context)
			}
		}

		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Add("Cache-Control", "no-cache")
		w.Header().Add(LongPollEtag, newEtag)
		w.Header().Add("X-Total-Count", fmt.Sprint(context["total"]))
		routes.ServeJson(w, context["response"])
	}
	route.Get(pluralURL, middleware.Authorization(schema.ActionRead), getPluralFunc)
	route.Get(pluralURLWithParents, middleware.Authorization(schema.ActionRead), func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addParamToQuery(r, schema.FormatParentID(s.Parent), p[s.Parent])
		getPluralFunc(w, r, p, identityService, context)
	})

	//setup show route
	getSingleFunc := func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addJSONContentTypeHeader(w)
		fillInContext(context, dataStore, r, w, s, p, server.sync, identityService, server.queue)
		id := p["id"]

		var newEtag string
		var err error
		if r.Header.Get(LongPollHeader) != "" {
			oldEtag := r.Header.Get(LongPollHeader)
			getResource := func(context middleware.Context) error {
				return resources.GetSingleResource(context, dataStore, s, id)
			}
			newEtag, err = server.longPoll.GetOrWait(r.URL.Path, oldEtag, context, getResource, calculateResponseEtag)
		} else {
			err = resources.GetSingleResource(context, dataStore, s, id)
			if err == nil {
				newEtag = calculateResponseEtag(context)
			}
		}

		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Add("Cache-Control", "no-cache, no-store")
		w.Header().Add(LongPollEtag, newEtag)
		routes.ServeJson(w, context["response"])
	}
	route.Get(singleURL, middleware.Authorization(schema.ActionRead), getSingleFunc)
	route.Get(singleURLWithParents, middleware.Authorization(schema.ActionRead), func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addParamToQuery(r, schema.FormatParentID(s.Parent), p[s.Parent])
		getSingleFunc(w, r, p, identityService, context)
	})

	//setup delete route
	deleteSingleFunc := func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addJSONContentTypeHeader(w)
		fillInContext(context, dataStore, r, w, s, p, server.sync, identityService, server.queue)
		id := p["id"]
		if err := resources.DeleteResource(context, dataStore, s, id); err != nil {
			handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	route.Delete(singleURL, middleware.Authorization(schema.ActionDelete), deleteSingleFunc)
	route.Delete(singleURLWithParents, middleware.Authorization(schema.ActionDelete), func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addParamToQuery(r, schema.FormatParentID(s.Parent), p[s.Parent])
		deleteSingleFunc(w, r, p, identityService, context)
	})

	//setup create route
	postPluralFunc := func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addJSONContentTypeHeader(w)
		fillInContext(context, dataStore, r, w, s, p, server.sync, identityService, server.queue)
		dataMap, err := middleware.ReadJSON(r)
		if err != nil {
			handleError(w, resources.NewResourceError(err, fmt.Sprintf("Failed to parse data: %s", err), resources.WrongData))
			return
		}
		dataMap = removeResourceWrapper(s, dataMap)
		if s.Parent != "" {
			if _, ok := dataMap[s.ParentID()]; !ok {
				queryParams := r.URL.Query()
				parentIDParam := queryParams.Get(s.ParentID())
				if parentIDParam != "" {
					dataMap[s.ParentID()] = parentIDParam
				}
			}
		}
		if err := resources.CreateResource(context, dataStore, identityService, s, dataMap); err != nil {
			handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
		routes.ServeJson(w, context["response"])
	}
	route.Post(pluralURL, middleware.Authorization(schema.ActionCreate), postPluralFunc)
	route.Post(pluralURLWithParents, middleware.Authorization(schema.ActionCreate),
		func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
			addParamToQuery(r, schema.FormatParentID(s.Parent), p[s.Parent])
			postPluralFunc(w, r, p, identityService, context)
		})

	//setup create or update route
	putSingleFunc := func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addJSONContentTypeHeader(w)
		fillInContext(context, dataStore, r, w, s, p, server.sync, identityService, server.queue)
		id := p["id"]
		dataMap, err := middleware.ReadJSON(r)
		if err != nil {
			handleError(w, resources.NewResourceError(err, fmt.Sprintf("Failed to parse data: %s", err), resources.WrongData))
			return
		}
		dataMap = removeResourceWrapper(s, dataMap)
		if isCreated, err := resources.CreateOrUpdateResource(
			context, dataStore, identityService, s, id, dataMap); err != nil {
			handleError(w, err)
			return
		} else if isCreated {
			w.WriteHeader(http.StatusCreated)
		}
		routes.ServeJson(w, context["response"])
	}
	route.Put(singleURL, middleware.Authorization(schema.ActionUpdate), putSingleFunc)
	route.Put(singleURLWithParents, middleware.Authorization(schema.ActionUpdate),
		func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
			addParamToQuery(r, schema.FormatParentID(s.Parent), p[s.Parent])
			putSingleFunc(w, r, p, identityService, context)
		})

	//setup update route
	patchSingleFunc := func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
		addJSONContentTypeHeader(w)
		fillInContext(context, dataStore, r, w, s, p, server.sync, identityService, server.queue)
		id := p["id"]
		dataMap, err := middleware.ReadJSON(r)
		if err != nil {
			handleError(w, resources.NewResourceError(err, fmt.Sprintf("Failed to parse data: %s", err), resources.WrongData))
			return
		}
		dataMap = removeResourceWrapper(s, dataMap)
		if err := resources.UpdateResource(
			context, dataStore, identityService, s, id, dataMap); err != nil {
			handleError(w, err)
			return
		}
		routes.ServeJson(w, context["response"])
	}
	route.Patch(singleURL, middleware.Authorization(schema.ActionUpdate), patchSingleFunc)
	route.Patch(singleURLWithParents, middleware.Authorization(schema.ActionUpdate),
		func(w http.ResponseWriter, r *http.Request, p martini.Params, identityService middleware.IdentityService, context middleware.Context) {
			addParamToQuery(r, schema.FormatParentID(s.Parent), p[s.Parent])
			patchSingleFunc(w, r, p, identityService, context)
		})

	//Custom action support
	for _, actionExt := range s.Actions {
		action := actionExt
		ActionFunc := func(w http.ResponseWriter, r *http.Request, p martini.Params,
			identityService middleware.IdentityService, auth schema.Authorization, context middleware.Context) {
			addJSONContentTypeHeader(w)
			fillInContext(context, dataStore, r, w, s, p, server.sync, identityService, server.queue)
			id := p["id"]
			input := make(map[string]interface{})
			if action.InputSchema != nil {
				var err error
				input, err = middleware.ReadJSON(r)
				if err != nil {
					handleError(w, resources.NewResourceError(err, fmt.Sprintf("Failed to parse data: %s", err), resources.WrongData))
					return
				}
			}

			// TODO use authorization middleware
			manager := schema.GetManager()
			path := r.URL.Path
			policy, role := manager.PolicyValidate(action.ID, path, auth)
			if policy == nil {
				middleware.HTTPJSONError(w, fmt.Sprintf("No matching policy: %s %s %s", action, path, s.Actions), http.StatusUnauthorized)
				return
			}
			context["policy"] = policy
			context["tenant_id"] = auth.TenantID()
			context["auth_token"] = auth.AuthToken()
			context["role"] = role
			context["catalog"] = auth.Catalog()
			context["auth"] = auth

			if err := resources.ActionResource(
				context, dataStore, identityService, s, action, id, input); err != nil {
				handleError(w, err)
				return
			}
			routes.ServeJson(w, context["response"])
		}
		route.AddRoute(action.Method, s.GetActionURL(action.Path), ActionFunc)
	}
}

//MapRouteBySchemas setup route for all loaded schema
func MapRouteBySchemas(server *Server, dataStore db.DB) {
	route := server.martini
	log.Debug("[Initializing Routes]")
	schemaManager := schema.GetManager()
	route.Get("/_all", func(w http.ResponseWriter, r *http.Request, p martini.Params, auth schema.Authorization) {
		responses := make(map[string]interface{})
		context := map[string]interface{}{
			"path":          r.URL.Path,
			"http_request":  r,
			"http_response": w,
		}
		for _, s := range schemaManager.Schemas() {
			policy, role := authorization(w, r, schema.ActionRead, s.GetPluralURL(), s, auth)
			if policy == nil {
				continue
			}
			context["policy"] = policy
			context["role"] = role
			context["auth"] = auth
			context["sync"] = server.sync
			if err := resources.GetResources(
				context, dataStore,
				s,
				resources.FilterFromQueryParameter(
					s, r.URL.Query()), nil); err != nil {
				handleError(w, err)
				return
			}
			resources.ApplyPolicyForResources(context, s)
			response := context["response"].(map[string]interface{})
			responses[s.GetDbTableName()] = response[s.Plural]
		}
		routes.ServeJson(w, responses)
	})
	for _, s := range schemaManager.Schemas() {
		MapRouteBySchema(server, dataStore, s)
	}
}

// MapNamespacesRoutes maps routes for all namespaces
func MapNamespacesRoutes(route martini.Router) {
	manager := schema.GetManager()

	for _, namespace := range manager.Namespaces() {
		if namespace.IsTopLevel() {
			mapTopLevelNamespaceRoute(route, namespace)
		} else {
			mapChildNamespaceRoute(route, namespace)
		}
	}
}

// mapTopLevelNamespaceRoute maps route listing available subnamespaces (versions)
// for a top-level namespace
func mapTopLevelNamespaceRoute(route martini.Router, namespace *schema.Namespace) {
	log.Debug("[Path] %s/", namespace.GetFullPrefix())
	route.Get(
		namespace.GetFullPrefix()+"/",
		func(w http.ResponseWriter, r *http.Request, p martini.Params, context martini.Context) {
			versions := []schema.Version{}
			for _, childNamespace := range schema.GetManager().Namespaces() {
				if childNamespace.Parent == namespace.ID {
					versions = append(versions, schema.Version{
						Status: "SUPPORTED",
						ID:     childNamespace.Prefix,
						Links: []schema.Link{
							schema.Link{
								Href: childNamespace.GetFullPrefix() + "/",
								Rel:  "self",
							},
						},
					})
				}
			}

			if len(versions) != 0 {
				versions[len(versions)-1].Status = "CURRENT"
			}

			routes.ServeJson(w, map[string][]schema.Version{"versions": versions})
		})
}

// mapChildNamespaceRoute sets a handler returning a dictionary of resources
// supported by a certain API version identified by the given namespace
func mapChildNamespaceRoute(route martini.Router, namespace *schema.Namespace) {
	log.Debug("[Path] %s", namespace.GetFullPrefix())
	route.Get(
		namespace.GetFullPrefix(),
		func(w http.ResponseWriter, r *http.Request, p martini.Params, context martini.Context) {
			resources := []schema.NamespaceResource{}
			for _, s := range schema.GetManager().Schemas() {
				if s.NamespaceID == namespace.ID {
					resources = append(resources, schema.NamespaceResource{
						Links: []schema.Link{
							schema.Link{
								Href: s.GetPluralURL(),
								Rel:  "self",
							},
						},
						Name:       s.Singular,
						Collection: s.Plural,
					})
				}
			}

			routes.ServeJson(w, map[string][]schema.NamespaceResource{"resources": resources})
		},
	)
}
