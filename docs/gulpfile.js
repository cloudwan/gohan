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


var gulp = require('gulp')
var shell = require('gulp-shell')
var browserSync = require('browser-sync');


gulp.task('browser-sync', function() {
  browserSync({
    server: {
      baseDir: './build/html'
    },
    port: 18000
  });
});

gulp.task('bs-reload', function() {
  browserSync.reload();
});


gulp.task('build-docs',  shell.task('make html',  {cwd: '.'}))

gulp.task('default',  ['browser-sync'],  function() {
  gulp.watch(['./source/*.rst', './source/*.py'],  ['build-docs']);
  gulp.watch(['./build/html/*'],  ['bs-reload']);
})
