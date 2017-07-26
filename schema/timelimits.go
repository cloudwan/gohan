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
	"regexp"
	"time"
)

// PathEventTimeLimit is a configuration for
// 	time limits for a regex path and a regex event
type PathEventTimeLimit struct {
	PathRegex    *regexp.Regexp
	EventRegex   *regexp.Regexp
	TimeDuration time.Duration
}

// NewPathEventTimeLimit is a constructor for PathEventTimeLimit
func NewPathEventTimeLimit(pathRegex, eventRegex string, timeDuration time.Duration) *PathEventTimeLimit {
	return &PathEventTimeLimit{
		PathRegex:    regexp.MustCompile(pathRegex),
		EventRegex:   regexp.MustCompile(eventRegex),
		TimeDuration: timeDuration,
	}
}

//Match checks if this path matches for extension
func (pathEventTimeLimit *PathEventTimeLimit) Match(path string) bool {
	return pathEventTimeLimit.PathRegex.MatchString(path)
}

// EventTimeLimit is a configuration for
// 	time limits for a regex event
type EventTimeLimit struct {
	EventRegex   *regexp.Regexp
	TimeDuration time.Duration
}

// NewEventTimeLimit is a constructor for EventTimeLimit
func NewEventTimeLimit(eventRegex *regexp.Regexp, timeLimit time.Duration) *EventTimeLimit {
	return &EventTimeLimit{
		EventRegex:   eventRegex,
		TimeDuration: timeLimit,
	}
}

//Match checks if this path matches for extension
func (eventTimeLimit *EventTimeLimit) Match(event string) bool {
	return eventTimeLimit.EventRegex.MatchString(event)
}
