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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	// maxKeySniffSize skips big files: private keys are a few KB at most.
	maxKeySniffSize = 64 * 1024
	// keygenTimeout bounds the ssh-keygen fingerprint lookup.
	keygenTimeout = 2 * time.Second

	keyPickerHint = "[white]Enter[-] Use as only key  • [white]+[-] Add to keys  • [white]r[-] ~/.ssh ↔ ~  • " +
		"[white]a[-] All files  • [white].[-] Hidden  • [white]Esc[-] Cancel"
)

// looksLikePrivateKey reports whether path is a regular file whose first line is a
// PEM/OpenSSH private key header. Only the header line is read, never the key material.
func looksLikePrivateKey(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxKeySniffSize {
		return false
	}
	// #nosec G304 - path comes from directory enumeration of the picker
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	buf := make([]byte, 64)
	n, _ := file.Read(buf)
	first, _, _ := bytes.Cut(buf[:n], []byte("\n"))
	return bytes.HasPrefix(first, []byte("-----BEGIN")) && bytes.Contains(first, []byte("PRIVATE KEY"))
}

// tildePath shortens a path under the home directory to the "~/..." form used in ssh configs.
func tildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	if rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}

// expandTilde resolves a leading "~" to the home directory.
func expandTilde(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

// mergeKeys returns the new value of the Keys field after picking key.
// With add=false the picked key replaces the field; with add=true it is appended (no duplicates).
func mergeKeys(current, picked string, add bool) string {
	if !add {
		return picked
	}
	var keys []string
	for _, k := range strings.Split(current, ",") {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		if k == picked {
			return strings.Join(keys, ", ")
		}
	}
	return strings.Join(append(keys, picked), ", ")
}

// keyFingerprint describes a key through ssh-keygen. The .pub sibling is preferred, and an
// empty passphrase is always passed so ssh-keygen can never prompt on the terminal.
func keyFingerprint(ctx context.Context, path string) (string, error) {
	target := path
	if !strings.HasSuffix(path, ".pub") {
		if st, err := os.Stat(path + ".pub"); err == nil && st.Mode().IsRegular() {
			target = path + ".pub"
		}
	}
	ctx, cancel := context.WithTimeout(ctx, keygenTimeout)
	defer cancel()
	// #nosec G204 - fixed binary; target is passed as a single argument, not through a shell
	out, err := exec.CommandContext(ctx, "ssh-keygen", "-l", "-f", target, "-P", "").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type pickerNode struct {
	path  string
	isDir bool
}

// KeyPicker is a full-screen file browser that lists private keys (and folders) so a host
// can be bound to a specific key without typing its path.
type KeyPicker struct {
	*tview.Flex
	app  *tview.Application
	tree *tview.TreeView
	info *tview.TextView

	root       string
	showAll    bool
	showHidden bool
	seq        int // incremented on every highlight; drops stale fingerprint results

	onPick   func(path string, add bool)
	onCancel func()
}

// NewKeyPicker builds the picker rooted at startDir (falls back to ~/.ssh, then ~).
// onPick receives the path in "~/..." form; add=true means "append to the existing keys".
func NewKeyPicker(app *tview.Application, startDir string, onPick func(string, bool), onCancel func()) *KeyPicker {
	kp := &KeyPicker{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		app:      app,
		tree:     tview.NewTreeView(),
		info:     tview.NewTextView().SetDynamicColors(true),
		onPick:   onPick,
		onCancel: onCancel,
	}
	kp.tree.SetBorder(true).SetTitle(" Pick SSH key ").SetTitleAlign(tview.AlignCenter)
	kp.tree.SetGraphics(true).SetTopLevel(0)
	kp.tree.SetChangedFunc(kp.describe)
	kp.tree.SetSelectedFunc(kp.selected)
	kp.info.SetBorder(true).SetTitle(" Key ").SetBorderPadding(0, 0, 1, 1)

	hint := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	hint.SetBackgroundColor(tcell.Color235)
	hint.SetText(keyPickerHint)

	kp.AddItem(kp.tree, 0, 1, true).AddItem(kp.info, 4, 0, false).AddItem(hint, 1, 0, false)
	kp.tree.SetInputCapture(kp.handleKey)

	kp.setRoot(kp.initialRoot(startDir))
	return kp
}

func (kp *KeyPicker) initialRoot(startDir string) string {
	if st, err := os.Stat(startDir); err == nil && st.IsDir() {
		return startDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	if sshDir := filepath.Join(home, ".ssh"); isDir(sshDir) {
		return sshDir
	}
	return home
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func (kp *KeyPicker) setRoot(dir string) {
	kp.root = dir
	rootNode := tview.NewTreeNode(tview.Escape(tildePath(dir))).
		SetReference(&pickerNode{path: dir, isDir: true}).
		SetColor(tcell.ColorYellow)
	kp.loadChildren(rootNode)
	rootNode.SetExpanded(true)
	kp.tree.SetRoot(rootNode)
	if children := rootNode.GetChildren(); len(children) > 0 {
		kp.tree.SetCurrentNode(children[0])
	} else {
		kp.tree.SetCurrentNode(rootNode)
	}
	kp.describe(kp.tree.GetCurrentNode())
}

// reload rebuilds the tree after a filter toggle, keeping the current root.
func (kp *KeyPicker) reload() { kp.setRoot(kp.root) }

func (kp *KeyPicker) loadChildren(node *tview.TreeNode) {
	ref, ok := node.GetReference().(*pickerNode)
	if !ok || !ref.isDir {
		return
	}
	node.ClearChildren()
	entries, err := os.ReadDir(ref.path)
	if err != nil {
		node.AddChild(tview.NewTreeNode("[red](cannot read folder)[-]").SetSelectable(false))
		return
	}
	sort.SliceStable(entries, func(i, j int) bool {
		di, dj := isDir(filepath.Join(ref.path, entries[i].Name())), isDir(filepath.Join(ref.path, entries[j].Name()))
		return di && !dj
	})
	for _, e := range entries {
		name := e.Name()
		if !kp.showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(ref.path, name)
		switch {
		case isDir(full):
			node.AddChild(tview.NewTreeNode(tview.Escape(name) + "/").
				SetReference(&pickerNode{path: full, isDir: true}).
				SetColor(tcell.ColorSteelBlue).
				SetExpanded(false))
		case kp.showAll || looksLikePrivateKey(full):
			node.AddChild(tview.NewTreeNode(tview.Escape(name)).
				SetReference(&pickerNode{path: full}).
				SetColor(tcell.ColorGreen))
		}
	}
	if len(node.GetChildren()) == 0 {
		node.AddChild(tview.NewTreeNode("[gray](no keys here - 'a' shows all files)[-]").SetSelectable(false))
	}
}

func (kp *KeyPicker) selected(node *tview.TreeNode) {
	ref, ok := node.GetReference().(*pickerNode)
	if !ok {
		return
	}
	if ref.isDir {
		if node.IsExpanded() {
			node.Collapse()
			return
		}
		kp.loadChildren(node)
		node.Expand()
		return
	}
	kp.onPick(tildePath(ref.path), false)
}

func (kp *KeyPicker) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		kp.onCancel()
		return nil
	}
	if event.Key() != tcell.KeyRune {
		return event
	}
	switch event.Rune() {
	case '+':
		if ref, ok := kp.tree.GetCurrentNode().GetReference().(*pickerNode); ok && !ref.isDir {
			kp.onPick(tildePath(ref.path), true)
		}
		return nil
	case 'r':
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		if sshDir := filepath.Join(home, ".ssh"); kp.root != sshDir && isDir(sshDir) {
			kp.setRoot(sshDir)
		} else {
			kp.setRoot(home)
		}
		return nil
	case 'a':
		kp.showAll = !kp.showAll
		kp.reload()
		return nil
	case '.':
		kp.showHidden = !kp.showHidden
		kp.reload()
		return nil
	}
	return event
}

// describe fills the info box for the highlighted node; fingerprints are computed off the UI thread.
func (kp *KeyPicker) describe(node *tview.TreeNode) {
	kp.seq++
	seq := kp.seq
	ref, ok := node.GetReference().(*pickerNode)
	if !ok {
		kp.info.SetText("")
		return
	}
	if ref.isDir {
		kp.info.SetText(tview.Escape(tildePath(ref.path)) + "\n[gray]Enter opens/closes the folder[-]")
		return
	}
	kp.info.SetText(tview.Escape(tildePath(ref.path)) + "\n[gray]reading fingerprint...[-]")
	go func() {
		fp, err := keyFingerprint(context.Background(), ref.path)
		kp.app.QueueUpdateDraw(func() {
			if seq != kp.seq {
				return
			}
			text := tview.Escape(tildePath(ref.path)) + "\n"
			if err != nil {
				text += "[yellow]not recognized as an SSH key (or ssh-keygen unavailable)[-]"
			} else {
				text += "[green]" + tview.Escape(fp) + "[-]"
			}
			kp.info.SetText(text)
		})
	}()
}
