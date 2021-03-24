// Copyright 2017 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

// Event represents a registry update event.
type Event int

const (
	// EventAdd is sent when an object is added.
	EventAdd Event = iota

	// EventUpdate is sent when an object is modified
	// Captures the modified object.
	EventUpdate

	// EventDelete is sent when an object is deleted
	// Captures the object at the last known state.
	EventDelete
)

func (e Event) String() string {
	out := "unknown"

	switch e {
	case EventAdd:
		out = "add"
	case EventUpdate:
		out = "update"
	case EventDelete:
		out = "delete"
	}

	return out
}
