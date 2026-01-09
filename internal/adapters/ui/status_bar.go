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
	"fmt"

	"github.com/rivo/tview"
)

func DefaultStatusText() string {
	k := CurrentTheme.HintKey
	return fmt.Sprintf("[%s]↑↓[-] Navigate  • [%s]Enter[-] SSH  • [%s]f[-] Forward  • [%s]x[-] Stop  • [%s]c[-] Copy  • [%s]a[-] Add  • [%s]e[-] Edit  • [%s]g[-] Ping  • [%s]d[-] Del  • [%s]p[-] Pin  • [%s]T[-] Theme  • [%s]/[-] Search  • [%s]q[-] Quit",
		k, k, k, k, k, k, k, k, k, k, k, k, k)
}

func NewStatusBar() *tview.TextView {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBackgroundColor(CurrentTheme.StatusBarBackground)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText(DefaultStatusText())
	return status
}
