// Copyright 2025.
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

package domain

import (
	"fmt"
	"strings"
)

// ErrAmbiguousHost is returned when an alias is defined in more than one SSH
// config file (the main file plus one or more `Include`-d files) and the
// caller hasn't told the repository which file to write to.
//
// The TUI catches this error, prompts the user to pick a file, and re-invokes
// the operation with Server.SourceFile set to the chosen path.
type ErrAmbiguousHost struct {
	Alias      string
	Candidates []string
}

func (e *ErrAmbiguousHost) Error() string {
	return fmt.Sprintf("alias %q is defined in multiple files: %s", e.Alias, strings.Join(e.Candidates, ", "))
}
