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

package i18n

import (
	_ "embed"
	"encoding/json"
	"os"
	"strings"
	"sync"
)

//go:embed locales/en.json
var enBytes []byte

//go:embed locales/zh-CN.json
var zhCNBytes []byte

var (
	messages    map[string]string
	once        sync.Once
	lang        string
	// defaultLang is set at build time via -ldflags:
	//   -X github.com/Adembc/lazyssh/internal/i18n.defaultLang=zh-CN
	defaultLang string
)

func initMessages() {
	// Priority: 1. env var LAZYSSH_LANG  2. compile-time defaultLang  3. "en"
	lang = os.Getenv("LAZYSSH_LANG")
	if lang == "" {
		lang = defaultLang
	}
	if lang == "" {
		lang = "en"
	}

	// Normalize: case-insensitive matching → standard form "zh-CN" or "en"
	lang = strings.ToLower(lang)
	switch lang {
	case "zh-cn", "zh", "cn", "chs", "chinese":
		lang = "zh-CN"
	default:
		lang = "en"
	}

	messages = make(map[string]string)

	// Load defaults based on language
	switch lang {
	case "zh-CN":
		if err := json.Unmarshal(zhCNBytes, &messages); err != nil {
			messages = make(map[string]string)
		}
	default:
		if err := json.Unmarshal(enBytes, &messages); err != nil {
			messages = make(map[string]string)
		}
	}
}

func Lang() string {
	once.Do(initMessages)
	return lang
}

// T returns the translated string for the given key.
// If the key is not found, it returns the key itself as fallback.
func T(key string) string {
	once.Do(initMessages)
	if msg, ok := messages[key]; ok {
		return msg
	}
	return key
}
