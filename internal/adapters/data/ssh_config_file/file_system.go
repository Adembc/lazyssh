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

package ssh_config_file

import (
	"io"
	"os"
)

// FileSystem interface for file operations to enable testing.
type FileSystem interface {
	Open(name string) (io.ReadCloser, error)
	Create(name string) (io.WriteCloser, error)
	Stat(name string) (os.FileInfo, error)
	IsNotExist(err error) bool
	Remove(file string) error
	Rename(file string, path string) error
	Chmod(path string, perms os.FileMode) error
	OpenFile(path string, i int, perms os.FileMode) (*os.File, error)
	ReadDir(dir string) ([]os.DirEntry, error)
}

// DefaultFileSystem implements FileSystem using standard os package.
type DefaultFileSystem struct{}

func (fs DefaultFileSystem) Open(name string) (io.ReadCloser, error) {
	// #nosec G304 -- the file path is controlled internally, not user-supplied
	return os.Open(name)
}

func (fs DefaultFileSystem) Create(name string) (io.WriteCloser, error) {
	// #nosec G304 -- the file path is controlled internally, not user-supplied
	return os.Create(name)
}

func (fs DefaultFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs DefaultFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (fs DefaultFileSystem) Remove(file string) error {
	return os.Remove(file)
}

func (fs DefaultFileSystem) Rename(file string, path string) error {
	return os.Rename(file, path)
}

func (fs DefaultFileSystem) Chmod(path string, perms os.FileMode) error {
	return os.Chmod(path, perms)
}

func (fs DefaultFileSystem) OpenFile(path string, i int, perms os.FileMode) (*os.File, error) {
	// #nosec G304 -- the file path is controlled internally, not user-supplied
	return os.OpenFile(path, i, perms)
}

func (fs DefaultFileSystem) ReadDir(dir string) ([]os.DirEntry, error) {
	return os.ReadDir(dir)
}
