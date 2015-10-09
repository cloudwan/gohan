#!/usr/bin/python
#
# Copyright (C) 2015 NTT Innovation Institute, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
# implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os


def read_file(path):
    result = []
    f = open(path)
    for line in f:
        result.append(line)
    f.close()
    return result


def write_file(path, lines):
    f = open(path, 'w')
    f.writelines(lines)
    f.close()


def startswith(content_a, content_b):
    l = len(content_a)
    if len(content_b) < l:
        return False
    for i in range(l):
        if content_a[i] != content_b[i]:
            return False
    return True


def find_files(directory, extension, ignore_dir):
    for root, dirs, files in os.walk(directory):
        if root.find(ignore_dir) > 0:
            continue
        for f in files:
            if f.endswith(extension):
                yield os.path.join(root, f)


license = read_file("./tools/go_license_header")

# process go files
for go_file in find_files(".", ".go", "Godeps"):
    go_code = read_file(go_file)
    if not startswith(license, go_code):
        print("%s has no license header. Adding.." % go_file)
        new_source = []
        new_source.extend(license)
        new_source.extend(['\n', '\n'])
        new_source.extend(go_code)
        write_file(go_file, new_source)

# process go files
for js_file in find_files(".", ".js", "webui"):
    js_code = read_file(js_file)
    if not startswith(license, js_code):
        print("%s has no license header. Adding.." % js_file)
        new_source = []
        new_source.extend(license)
        new_source.extend(['\n', '\n'])
        new_source.extend(js_code)
        write_file(js_file, new_source)
