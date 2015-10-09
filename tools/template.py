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

import argparse
import jinja2
import yaml

parser = argparse.ArgumentParser()
parser.add_argument('-s', '--schema', help="schema")
parser.add_argument('-t', '--template', help="template")

args = parser.parse_args()
schema = args.schema
template = args.template

f = open(schema, 'r')
schema_obj = yaml.load(f)
f.close()

templateLoader = jinja2.FileSystemLoader(searchpath=".")
templateEnv = jinja2.Environment(loader=templateLoader, line_statement_prefix='#')
template = templateEnv.get_template(template)

print(template.render(schema_obj))
