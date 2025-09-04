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

package flags

import (
	"github.com/Adembc/lazyssh/internal/core/ports"

	"github.com/spf13/cobra"
)

type CobraFlags struct {
	rootCmd *cobra.Command
}

func NewCobraFlags(rootCmd *cobra.Command) ports.FlagsProvider {
	g := &CobraFlags{rootCmd: rootCmd}
	g.globalFlags()
	return g
}

// registerFlags registers all global flags here
func (g *CobraFlags) globalFlags() {
	// Example: debug flag
	g.rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")

	// You can add more global flags here in future
}

func (c *CobraFlags) IsDebug() bool {
	flag, _ := c.rootCmd.Flags().GetBool("debug")
	return flag
}

func (c *CobraFlags) GetFlag(name string) string {
	value, _ := c.rootCmd.Flags().GetString(name)
	return value
}
