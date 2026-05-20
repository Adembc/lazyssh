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
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/i18n"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServerDetails struct {
	*tview.TextView
}

func NewServerDetails() *ServerDetails {
	details := &ServerDetails{
		TextView: tview.NewTextView(),
	}
	details.build()
	return details
}

func (sd *ServerDetails) build() {
	sd.TextView.SetDynamicColors(true).
		SetWrap(true).
		SetBorder(true).
		SetTitle(i18n.T("details.title")).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
}

// renderTagChips builds colored tag chips for details view.
func renderTagChips(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	chips := make([]string, 0, len(tags))
	for _, t := range tags {
		chips = append(chips, fmt.Sprintf("[black:#5FAFFF] %s [-:-:-]", t))
	}
	return strings.Join(chips, " ")
}

func (sd *ServerDetails) UpdateServer(server domain.Server) {
	lastSeen := server.LastSeen.Format("2006-01-02 15:04:05")
	if server.LastSeen.IsZero() {
		lastSeen = i18n.T("details.never")
	}
	serverKey := strings.Join(server.IdentityFiles, ", ")

	pinnedStr := "true"
	if server.PinnedAt.IsZero() {
		pinnedStr = "false"
	}
	tagsText := renderTagChips(server.Tags)

	// Basic information
	aliasText := strings.Join(server.Aliases, ", ")

	userText := server.User

	hostText := server.Host

	portText := fmt.Sprintf("%d", server.Port)
	if server.Port == 0 {
		portText = ""
	}

	text := fmt.Sprintf(
		"[::b]%s[-]\n\n[::b]%s[-]\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n",
		aliasText,
		i18n.T("details.label.basic"),
		fmt.Sprintf(i18n.T("details.host"), hostText),
		fmt.Sprintf(i18n.T("details.user"), userText),
		fmt.Sprintf(i18n.T("details.port"), portText),
		fmt.Sprintf(i18n.T("details.key"), serverKey),
		fmt.Sprintf(i18n.T("details.tags"), tagsText),
		fmt.Sprintf(i18n.T("details.pinned"), pinnedStr),
		fmt.Sprintf(i18n.T("details.last_ssh"), lastSeen),
		fmt.Sprintf(i18n.T("details.ssh_count"), server.SSHCount))

	// Advanced settings section (only show non-empty fields)
	// Organized by logical grouping for better readability
	type fieldEntry struct {
		name  string
		value string
	}

	type fieldGroup struct {
		name   string
		fields []fieldEntry
	}

	// Create field groups for better organization and future extensibility
	groups := []fieldGroup{
		{
			name: i18n.T("section_proxy_command"),
			fields: []fieldEntry{
				{"ProxyJump", server.ProxyJump},
				{"ProxyCommand", server.ProxyCommand},
				{"RemoteCommand", server.RemoteCommand},
				{"RequestTTY", server.RequestTTY},
				{"SessionType", server.SessionType},
				{"ConnectTimeout", server.ConnectTimeout},
				{"ConnectionAttempts", server.ConnectionAttempts},
				{"BindAddress", server.BindAddress},
				{"BindInterface", server.BindInterface},
				{"AddressFamily", server.AddressFamily},
				{"ExitOnForwardFailure", server.ExitOnForwardFailure},
				{"IPQoS", server.IPQoS},
				{"CanonicalizeHostname", server.CanonicalizeHostname},
				{"CanonicalDomains", server.CanonicalDomains},
				{"CanonicalizeFallbackLocal", server.CanonicalizeFallbackLocal},
				{"CanonicalizeMaxDots", server.CanonicalizeMaxDots},
				{"CanonicalizePermittedCNAMEs", server.CanonicalizePermittedCNAMEs},
				{"ServerAliveInterval", server.ServerAliveInterval},
				{"ServerAliveCountMax", server.ServerAliveCountMax},
				{"Compression", server.Compression},
				{"TCPKeepAlive", server.TCPKeepAlive},
				{"BatchMode", server.BatchMode},
				{"ControlMaster", server.ControlMaster},
				{"ControlPath", server.ControlPath},
				{"ControlPersist", server.ControlPersist},
			},
		},
		{
			name: i18n.T("section_security"),
			fields: []fieldEntry{
				{"PubkeyAuthentication", server.PubkeyAuthentication},
				{"PubkeyAcceptedAlgorithms", server.PubkeyAcceptedAlgorithms},
				{"HostbasedAcceptedAlgorithms", server.HostbasedAcceptedAlgorithms},
				{"PasswordAuthentication", server.PasswordAuthentication},
				{"PreferredAuthentications", server.PreferredAuthentications},
				{"IdentitiesOnly", server.IdentitiesOnly},
				{"AddKeysToAgent", server.AddKeysToAgent},
				{"IdentityAgent", server.IdentityAgent},
				{"KbdInteractiveAuthentication", server.KbdInteractiveAuthentication},
				{"NumberOfPasswordPrompts", server.NumberOfPasswordPrompts},
			},
		},
		{
			name: i18n.T("section_port_forwarding"),
			fields: []fieldEntry{
				{"ForwardAgent", server.ForwardAgent},
				{"ForwardX11", server.ForwardX11},
				{"ForwardX11Trusted", server.ForwardX11Trusted},
				{"LocalForward", strings.Join(server.LocalForward, ", ")},
				{"RemoteForward", strings.Join(server.RemoteForward, ", ")},
				{"DynamicForward", strings.Join(server.DynamicForward, ", ")},
				{"ClearAllForwardings", server.ClearAllForwardings},
				{"GatewayPorts", server.GatewayPorts},
			},
		},
		{
			name: i18n.T("section_security"),
			fields: []fieldEntry{
				{"StrictHostKeyChecking", server.StrictHostKeyChecking},
				{"CheckHostIP", server.CheckHostIP},
				{"FingerprintHash", server.FingerprintHash},
				{"UserKnownHostsFile", server.UserKnownHostsFile},
				{"HostKeyAlgorithms", server.HostKeyAlgorithms},
				{"Ciphers", server.Ciphers},
				{"MACs", server.MACs},
				{"KexAlgorithms", server.KexAlgorithms},
				{"VerifyHostKeyDNS", server.VerifyHostKeyDNS},
				{"UpdateHostKeys", server.UpdateHostKeys},
				{"HashKnownHosts", server.HashKnownHosts},
				{"VisualHostKey", server.VisualHostKey},
			},
		},
		{
			name: i18n.T("section_environment"),
			fields: []fieldEntry{
				{"LocalCommand", server.LocalCommand},
				{"PermitLocalCommand", server.PermitLocalCommand},
				{"EscapeChar", server.EscapeChar},
				{"SendEnv", strings.Join(server.SendEnv, ", ")},
				{"SetEnv", strings.Join(server.SetEnv, ", ")},
			},
		},
		{
			name: i18n.T("section_debugging"),
			fields: []fieldEntry{
				{"LogLevel", server.LogLevel},
			},
		},
	}

	// Build advanced settings text without group labels for cleaner display
	hasAdvanced := false
	advancedText := "\n[::b]" + i18n.T("detail_advanced") + ":[-]\n"

	for _, group := range groups {
		for _, field := range group.fields {
			if field.value != "" {
				hasAdvanced = true
				advancedText += fmt.Sprintf("  [green]%s:[white] %s[-]\n", translateDetailFieldName(field.name), field.value)
			}
		}
	}

	if hasAdvanced {
		text += advancedText
	}

	// Commands list
	text += "\n[::b]" + i18n.T("detail_commands") + ":[-]\n" + i18n.T("detail_commands_list")

	sd.TextView.SetText(text)
}

func (sd *ServerDetails) ShowEmpty() {
	sd.TextView.SetText(i18n.T("details.no_match"))
}

// detailFieldNames maps raw ssh config field names to localized Chinese labels
var detailFieldNames = map[string]string{
	// Connection
	"ProxyJump":                   "跳板机",
	"ProxyCommand":                "代理命令",
	"RemoteCommand":               "远程命令",
	"RequestTTY":                  "请求TTY",
	"SessionType":                 "会话类型",
	"ConnectTimeout":              "连接超时",
	"ConnectionAttempts":          "连接重试次数",
	"BindAddress":                 "绑定地址",
	"BindInterface":               "绑定接口",
	"AddressFamily":               "地址族",
	"ExitOnForwardFailure":        "转发失败退出",
	"IPQoS":                       "IP QoS",
	"CanonicalizeHostname":        "主机名规范化",
	"CanonicalDomains":            "规范域名",
	"CanonicalizeFallbackLocal":   "规范回退",
	"CanonicalizeMaxDots":         "规范最大点数",
	"CanonicalizePermittedCNAMEs": "规范CNAME",
	"ServerAliveInterval":         "服务器保活间隔",
	"ServerAliveCountMax":         "保活尝试次数",
	"Compression":                 "压缩",
	"TCPKeepAlive":                "TCP保活",
	"BatchMode":                   "批处理模式",
	// Multiplexing
	"ControlMaster":  "主控连接",
	"ControlPath":    "控制路径",
	"ControlPersist": "连接持续",
	// Authentication
	"PubkeyAuthentication":      "公钥认证",
	"PasswordAuthentication":    "密码认证",
	"PreferredAuthentications":  "认证方式优先级",
	"IdentitiesOnly":            "仅身份文件",
	"AddKeysToAgent":            "添加密钥到代理",
	"IdentityAgent":             "身份代理",
	"KbdInteractiveAuthentication": "键盘交互认证",
	"NumberOfPasswordPrompts":   "密码提示次数",
	"PubkeyAcceptedAlgorithms":  "公钥接受算法",
	"HostbasedAcceptedAlgorithms": "基于主机的接受算法",
	// Forwarding
	"ForwardAgent":        "转发代理",
	"ForwardX11":          "X11转发",
	"ForwardX11Trusted":   "X11转发(信任)",
	"LocalForward":        "本地转发",
	"RemoteForward":       "远程转发",
	"DynamicForward":      "动态转发",
	"ClearAllForwardings": "清除所有转发",
	"GatewayPorts":        "网关端口",
	// Security
	"StrictHostKeyChecking": "严格主机密钥检查",
	"CheckHostIP":           "检查主机IP",
	"FingerprintHash":      "指纹哈希",
	"UserKnownHostsFile":   "已知主机文件",
	"HostKeyAlgorithms":     "主机密钥算法",
	"Ciphers":               "加密算法",
	"MACs":                  "MAC算法",
	"KexAlgorithms":         "密钥交换算法",
	"VerifyHostKeyDNS":      "DNS验证主机密钥",
	"UpdateHostKeys":        "更新主机密钥",
	"HashKnownHosts":        "哈希已知主机",
	"VisualHostKey":         "可视化主机密钥",
	// Environment
	"LocalCommand":       "本地命令",
	"PermitLocalCommand": "允许本地命令",
	"EscapeChar":         "转义字符",
	"SendEnv":            "发送环境变量",
	"SetEnv":             "设置环境变量",
	// Debug
	"LogLevel": "日志级别",
}

func translateDetailFieldName(name string) string {
	if i18n.Lang() != "zh-CN" {
		return name
	}
	if zh, ok := detailFieldNames[name]; ok {
		return zh
	}
	return name
}

