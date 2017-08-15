// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ygot

// GoStruct is an interface which can be implemented by Go structs that are
// generated to represent a YANG container or list member. It simply allows
// handling code to ensure that it is interacting with a struct that will meet
// the expectations of the interface - such as the fields being tagged with
// appropriate metadata (tags) that allow mapping of the struct into a YANG
// schematree.
type GoStruct interface {
	// IsYANGGoStruct is a marker method that indicates that the struct
	// implements the GoStruct interface.
	IsYANGGoStruct()
}

// ValidatedGoStruct is an interface which can be implemented by Go structs
// that are generated to represent a YANG container or list member that have
// the corresponding function to be validated against the a YANG schema.
type ValidatedGoStruct interface {
	GoStruct // Embed GoStruct since a ValidatedGoStruct must be a GoStruct.
	// Validate compares the contents of the implementing struct against
	// the YANG schema, and returns an error if the struct's contents
	// are not valid, or nil if the struct complies with the schema.
	Validate() error
}

// GoEnum is an interface which can be implemented by derived types which
// represent an enumerated value within a YANG schema. This allows handling
// code that finds struct fields that implement this interface to do specific
// mapping to other types when translating to a particular schematree.
type GoEnum interface {
	// IsYANGGoEnum is a marker method that indicates that the
	// struct implements the GoEnum interface.
	IsYANGGoEnum()
	// ΛMap is a method associated with each enumeration that retrieves a
	// map of the enumeration types to values that are associated with a
	// generated code file. The ygen library generates a static map of
	// enumeration values that this method returns.
	ΛMap() map[string]map[int64]string
}
