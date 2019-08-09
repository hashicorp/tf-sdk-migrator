package module

import (
	"fmt"
)

// A ModuleError indicates an error specific to a module.
type ModuleError struct {
	Path    string
	Version string
	Err     error
}

func (e *ModuleError) Error() string {
	if v, ok := e.Err.(*InvalidVersionError); ok {
		return fmt.Sprintf("%s@%s: invalid %s: %v", e.Path, v.Version, v.noun(), v.Err)
	}
	if e.Version != "" {
		return fmt.Sprintf("%s@%s: %v", e.Path, e.Version, e.Err)
	}
	return fmt.Sprintf("module %s: %v", e.Path, e.Err)
}

func (e *ModuleError) Unwrap() error { return e.Err }

// An InvalidVersionError indicates an error specific to a version, with the
// module path unknown or specified externally.
//
// A ModuleError may wrap an InvalidVersionError, but an InvalidVersionError
// must not wrap a ModuleError.
type InvalidVersionError struct {
	Version string
	Pseudo  bool
	Err     error
}

// noun returns either "version" or "pseudo-version", depending on whether
// e.Version is a pseudo-version.
func (e *InvalidVersionError) noun() string {
	if e.Pseudo {
		return "pseudo-version"
	}
	return "version"
}

func (e *InvalidVersionError) Error() string {
	return fmt.Sprintf("%s %q invalid: %s", e.noun(), e.Version, e.Err)
}

func (e *InvalidVersionError) Unwrap() error { return e.Err }
