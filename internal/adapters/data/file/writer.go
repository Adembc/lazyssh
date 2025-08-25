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

package file

import (
	"bufio"
	"fmt"
	"io"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

type SSHConfigWriter struct{}

func (w *SSHConfigWriter) Write(writer io.Writer, servers []domain.Server) error {
	bufWriter := bufio.NewWriter(writer)
	defer bufWriter.Flush()

	fmt.Fprintf(bufWriter, "%s\n\n", ManagedByComment)

	for i, server := range servers {
		if i > 0 {
			bufWriter.WriteString("\n")
		}
		w.writeServer(bufWriter, server)
	}

	return nil
}

func (w *SSHConfigWriter) writeServer(writer *bufio.Writer, server domain.Server) {
	fmt.Fprintf(writer, "Host %s\n", server.Alias)

	if server.Host != "" {
		fmt.Fprintf(writer, "    HostName %s\n", server.Host)
	}

	if server.User != "" {
		fmt.Fprintf(writer, "    User %s\n", server.User)
	}

	if server.Port != 0 && server.Port != DefaultPort {
		fmt.Fprintf(writer, "    Port %d\n", server.Port)
	}

	if server.Key != "" {
		fmt.Fprintf(writer, "    IdentityFile %s\n", server.Key)
	}
}
