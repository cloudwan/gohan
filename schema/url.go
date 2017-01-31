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

package schema

import (
	"fmt"
	"net/url"
	"path/filepath"
)

func isURLRelative(u *url.URL) bool {
	return (u.Scheme == "" || u.Scheme == "file") && (u.Host != "" || u.Host == "" && (u.Path == "" || u.Path[0:1] != "/"))
}

func fixRelativeURL(rawurl string, dir string) (string, error) {
	if rawurl == "" {
		return "", nil
	}

	u, err := url.Parse(rawurl)
	if err != nil {
		return "", fmt.Errorf("could not parse url: %s", err)
	}

	if u.Scheme == "" {
		u.Scheme = "file"
	}

	if !isURLRelative(u) {
		return u.String(), nil
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %s", err)
	}
	u.Host, u.Path = "", filepath.Join(dir, u.Host, u.Path)

	return u.String(), nil
}
