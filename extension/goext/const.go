// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

package goext

type ResourceEvent string
type CustomEvent string

// Built-in handler types
const (
	PreCreate    ResourceEvent = "pre_create"
	PreCreateTx  ResourceEvent = "pre_create_in_transaction"
	PostCreateTx ResourceEvent = "post_create_in_transaction"
	PostCreate   ResourceEvent = "post_create"

	PreUpdate    ResourceEvent = "pre_update"
	PreUpdateTx  ResourceEvent = "pre_update_in_transaction"
	PostUpdateTx ResourceEvent = "post_update_in_transaction"
	PostUpdate   ResourceEvent = "post_update"

	PreShow    ResourceEvent = "pre_show"
	PreShowTx  ResourceEvent = "pre_show_in_transaction"
	PostShowTx ResourceEvent = "post_show_in_transaction"
	PostShow   ResourceEvent = "post_show"

	PreList    ResourceEvent = "pre_list"
	PreListTx  ResourceEvent = "pre_list_in_transaction"
	PostListTx ResourceEvent = "post_list_in_transaction"
	PostList   ResourceEvent = "post_list"

	PreDelete    ResourceEvent = "pre_delete"
	PreDeleteTx  ResourceEvent = "pre_delete_in_transaction"
	PostDeleteTx ResourceEvent = "post_delete_in_transaction"
	PostDelete   ResourceEvent = "post_delete"
)

const (
	KeyTopLevelHandler = "top_level_handler"
)
