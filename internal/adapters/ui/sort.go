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

package ui

import (
	"sort"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

// SortMode controls how unpinned servers are ordered in the UI.
type SortMode int

const (
	SortByAliasAsc SortMode = iota
	SortByAliasDesc
	SortByLastSeenDesc
	SortByLastSeenAsc
)

func (m SortMode) String() string {
	switch m {
	case SortByAliasAsc:
		return "Alias ↑"
	case SortByAliasDesc:
		return "Alias ↓"
	case SortByLastSeenAsc:
		return "Last SSH ↑"
	case SortByLastSeenDesc:
		return "Last SSH ↓"
	default:
		return "Alias ↑"
	}
}

// ToggleField switches between Alias and LastSeen while preserving direction.
func (m SortMode) ToggleField() SortMode {
	switch m {
	case SortByAliasAsc:
		return SortByLastSeenAsc
	case SortByAliasDesc:
		return SortByLastSeenDesc
	case SortByLastSeenAsc:
		return SortByAliasAsc
	case SortByLastSeenDesc:
		return SortByAliasDesc
	default:
		return SortByAliasAsc
	}
}

// Reverse flips the direction within the current field.
func (m SortMode) Reverse() SortMode {
	switch m {
	case SortByAliasAsc:
		return SortByAliasDesc
	case SortByAliasDesc:
		return SortByAliasAsc
	case SortByLastSeenAsc:
		return SortByLastSeenDesc
	case SortByLastSeenDesc:
		return SortByLastSeenAsc
	default:
		return SortByAliasAsc
	}
}

// sortServersForUI sorts servers according to the rules required by the UI.
// Pinned servers are always at the top, ordered by pinned date (newest first).
// Unpinned servers are sorted by the selected mode. "Never" (zero time) goes to
// the bottom when sorting by last seen asc/desc accordingly. Ties break by Alias asc.
func sortServersForUI(servers []domain.Server, mode SortMode) {
	sort.SliceStable(servers, func(i, j int) bool {
		si, sj := servers[i], servers[j]

		pi, pj := !si.PinnedAt.IsZero(), !sj.PinnedAt.IsZero()
		if pi != pj {
			return pi
		}
		if pi && pj { // both pinned: newer pinned first, tie-break alias
			if !si.PinnedAt.Equal(sj.PinnedAt) {
				return si.PinnedAt.After(sj.PinnedAt)
			}
			return strings.ToLower(si.Alias) < strings.ToLower(sj.Alias)
		}

		// both unpinned
		switch mode {
		case SortByLastSeenDesc, SortByLastSeenAsc:
			zi := si.LastSeen.IsZero()
			zj := sj.LastSeen.IsZero()
			if zi != zj {
				// when sorting by last seen, entries with zero (never) should be bottom in either direction
				return !zi // non-zero first
			}
			if !zi && !zj && !si.LastSeen.Equal(sj.LastSeen) {
				if mode == SortByLastSeenDesc {
					return si.LastSeen.After(sj.LastSeen)
				}
				return si.LastSeen.Before(sj.LastSeen)
			}
			// tie-break by alias asc
			return strings.ToLower(si.Alias) < strings.ToLower(sj.Alias)
		case SortByAliasAsc:
			return strings.ToLower(si.Alias) < strings.ToLower(sj.Alias)
		case SortByAliasDesc:
			ai := strings.ToLower(si.Alias)
			aj := strings.ToLower(sj.Alias)
			if ai != aj {
				return ai > aj
			}
			return false
		default:
			return strings.ToLower(si.Alias) < strings.ToLower(sj.Alias)
		}
	})
}
