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

package ygen

import (
	"bytes"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// children returns all child elements of a directory element e that are not
// RPC entries.
func children(e *yang.Entry) []*yang.Entry {
	var entries []*yang.Entry

	for _, e := range e.Dir {
		if e.RPC == nil {
			entries = append(entries, e)
		}
	}
	return entries
}

// hasOnlyChild returns true if the directory passed to it only has a single
// element below it.
func hasOnlyChild(e *yang.Entry) bool {
	return e.Dir != nil && len(children(e)) == 1
}

// isRoot returns true if the entry is an entity at the root of the tree.
func isRoot(e *yang.Entry) bool {
	return e.Parent == nil
}

// isConfigState returns true if the entry is an entity that represents a
// container called config or state.
func isConfigState(e *yang.Entry) bool {
	return e.IsDir() && (e.Name == "config" || e.Name == "state")
}

// isChoiceOrCase returns true if the entry is either a 'case' or a 'choice'
// node within the schema. These are schema nodes only, and the code generation
// operates on data tree paths.
func isChoiceOrCase(e *yang.Entry) bool {
	return e.Kind == yang.CaseEntry || e.Kind == yang.ChoiceEntry
}

// isUnionType returns true if the entry is a union within the YANG schema,
// checked by determining the length of the Type slice within the YangType.
func isUnionType(t *yang.YangType) bool {
	return len(t.Type) > 0
}

// isEnumType returns true if the entry is an enumerated type within the
// YANG schema - i.e., an enumeration or identityref leaf.
func isEnumType(t *yang.YangType) bool {
	return t.Kind == yang.Yenum || t.Kind == yang.Yidentityref
}

// isOCCompressedValidElement returns true if the element would be output in the
// compressed YANG code.
func isOCCompressedValidElement(e *yang.Entry) bool {
	switch {
	case hasOnlyChild(e) && children(e)[0].IsList():
		// This is a surrounding container for a list which is removed from the
		// structure.
		return false
	case isRoot(e):
		// This is a top-level module within the goyang structure, so is not output.
		return false
	case isConfigState(e):
		// This is a container that is called config or state, which is removed from
		// a compressed OpenConfig schema.
		return false
	case isChoiceOrCase(e):
		// This is a choice or case node that is removed from the overall schema
		// so code generation does not occur for it.
		return false
	}
	return true
}

// joinPath concatenates a slice of strings into a / separated path. i.e.,
// []string{"", "foo", "bar"} becomes "/foo/bar". Paths in YANG are generally
// represented in this return format, but the []string format is more flexible
// for internal use.
func joinPath(parts []string) string {
	var buf bytes.Buffer
	for i, p := range parts {
		buf.WriteString(p)
		if i != len(parts)-1 {
			buf.WriteRune('/')
		}
	}
	return buf.String()
}

// findFirstNonChoice traverses the data tree and determines the first directory
// nodes from a root e that are neither case nor choice nodes. The map, m, is
// updated in place to append new entries that are found when recursively
// traversing the set of choice/case nodes.
func findFirstNonChoice(e *yang.Entry, m map[string]*yang.Entry) {
	switch {
	case !isChoiceOrCase(e):
		m[e.Path()] = e
	case e.IsDir():
		for _, ch := range e.Dir {
			findFirstNonChoice(ch, m)
		}
	}
}

// removePrefix verifies whether there is a ":" character in a path element
// and returns the unprefixed path if so. This is to handle cases where YANG
// XPath expressions may specify paths in the form prefix:pathelement.
func removePrefix(s string) string {
	if !strings.ContainsRune(s, ':') {
		return s
	}
	return strings.Split(s, ":")[1]
}

// parentModuleName returns the name of the module that defined the yang.Node
// supplied as the node argument. If the discovered root node of the node is found
// to be a submodule, the name of the parent module is returned.
func parentModuleName(node yang.Node) string {
	var definingMod yang.Node
	definingMod = yang.RootNode(node)
	if definingMod.Kind() == "submodule" {
		// A submodule must always be a *yang.Module.
		mod := definingMod.(*yang.Module)
		definingMod = mod.BelongsTo
	}

	if name, ok := camelCaseNameExt(definingMod.Exts()); ok {
		return name
	}

	return definingMod.NName()
}

// traverseElementSchemaPath takes an input yang.Entry and walks up the tree to find
// its path, expressed as a slice of strings, which is returned.
func traverseElementSchemaPath(elem *yang.Entry) []string {
	var pp []string
	e := elem
	for ; e.Parent != nil; e = e.Parent {
		if !isChoiceOrCase(e) {
			pp = append(pp, e.Name)
		}
	}
	pp = append(pp, e.Name)

	// Reverse the slice that was specified to us as it was appended to
	// from the leaf to the root.
	for i := len(pp)/2 - 1; i >= 0; i-- {
		o := len(pp) - 1 - i
		pp[i], pp[o] = pp[o], pp[i]
	}
	return pp
}

// isConfig takes a yang.Entry and traverses up the tree to find the config
// state of that element. In YANG, if the config parameter is unset, then it is
// is inherited from the parent of the element - hence we must walk up the tree to find
// the state. If the element at the top of the tree does not have config set, then config
// is true. See https://tools.ietf.org/html/rfc6020#section-7.19.1.
func isConfig(e *yang.Entry) bool {
	for ; e.Parent != nil; e = e.Parent {
		switch e.Config {
		case yang.TSTrue:
			return true
		case yang.TSFalse:
			return false
		}
	}

	// Reached the last element in the tree without explicit configuration
	// being set.
	if e.Config == yang.TSFalse {
		return false
	}
	return true
}

// slicePathToString takes a path represented as a slice of strings, and outputs
// it as a single string, with path elements separated by a forward slash.
func slicePathToString(path []string) string {
	var buf bytes.Buffer
	for i, elem := range path {
		buf.WriteString(elem)
		if i != len(path)-1 {
			buf.WriteRune('/')
		}
	}
	return buf.String()
}

// safeGoEnumeratedValueName takes an input string, which is the name of an
// enumerated value from a YANG schema, and ensures that it is safe to be
// output as part of the name of the enumerated value in the Go code. The
// sanitised value is returned.  Per RFC6020 Section 6.2, a YANG identifier is
// of the form [_a-zA-Z][a-zA-Z0-9\-\.]+ - such that we must replace "." and
// "-" characters.  The implementation used here replaces [\.\-] with "_"
// characters.  In OpenConfig schemas, there are currently a small number of
// identity values that contain "." and hence must be specifically handled.
func safeGoEnumeratedValueName(name string) string {
	// NewReplacer takes pairs of strings to be replaced in the form
	// old, new.
	replacer := strings.NewReplacer(
		".", "_",
		"-", "_",
		"/", "_",
		"+", "_PLUS",
		"*", "_ASTERISK",
		" ", "_")
	return replacer.Replace(name)
}

// instantiatingModule returns the name of the module that instantiates a node within
// the YANG schema tree. This is defined to be the node that 'uses' a grouping, contains
// an 'augment', or a module that directly defines the node.
func instantiatingModule(e *yang.Entry) string {
	// To be defined within the data tree, a node must either be:
	//  - Directly defined within a module - in which case, there is no "uses" or "augment"
	//    in the node hierarchy to it.
	//  - Defined in a grouping which is instantiated by a "uses" somewhere in the data tree -
	//    in which case the instantiating module is the module containing the "uses" statement.
	//  - Added to the schema tree through an augment, in which case the insantiating module is
	//    the module that contains the augment statement.
	//  Therefore, to find insantiating module for a particular entry, we walk up the node tree
	//  and determine the first 'instantiating' node type (uses, augment, module). We then return
	//  the root node that defined that entry, and use this as the defining module.
	n := e.Node
	for ; n.ParentNode() != nil && n.Kind() != "augment" && n.Kind() != "grouping"; n = n.ParentNode() {
	}

	if n.Kind() == "grouping" {
		for ; e.Parent != nil; e = e.Parent {
		}
		return yang.RootNode(e.Node).Name
	}

	// In a goyang parsed tree, a node cannot be nil since this is required to produce
	// the entry. All nodes must have a valid root (i.e., base node) which is the module
	// that they are defined in, and all nodes must have a name, therefore we can safely
	// just return the root node's name which corresponds to the name of the module within
	// which the node was defined.
	return yang.RootNode(n).Name
}

// enumeratedUnionTypes recursively searches the set of yang.YangTypes supplied to
// extract the enumerated types that are within a union.
func enumeratedUnionTypes(types []*yang.YangType) []*yang.YangType {
	var eTypes []*yang.YangType
	for _, t := range types {
		switch {
		case isEnumType(t):
			eTypes = append(eTypes, t)
		case isUnionType(t):
			eTypes = append(eTypes, enumeratedUnionTypes(t.Type)...)
		}
	}
	return eTypes
}

// appendIfNotEmpty appends a string s to a slice of strings if the string s is
// not nil, similarly to append it returns the modified slice.
func appendIfNotEmpty(slice []string, s string) []string {
	if s != "" {
		return append(slice, s)
	}
	return slice
}
