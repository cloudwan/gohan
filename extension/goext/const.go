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

// Built-in handler types
const (
	PreCreate    = "pre_create"
	PreCreateTx  = "pre_create_in_transaction"
	PostCreateTx = "post_create_in_transaction"
	PostCreate   = "post_create"

	PreUpdate    = "pre_update"
	PreUpdateTx  = "pre_update_in_transaction"
	PostUpdateTx = "post_update_in_transaction"
	PostUpdate   = "post_update"

	PreShow    = "pre_show"
	PreShowTx  = "pre_show_in_transaction"
	PostShowTx = "post_show_in_transaction"
	PostShow   = "post_show"

	PreList    = "pre_list"
	PreListTx  = "pre_list_in_transaction"
	PostListTx = "post_list_in_transaction"
	PostList   = "post_list"

	PreDelete    = "pre_delete"
	PreDeleteTx  = "pre_delete_in_transaction"
	PostDeleteTx = "post_delete_in_transaction"
	PostDelete   = "post_delete"
)
