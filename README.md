# lazyssh

**lazyssh** is a terminal-based, interactive SSH manager inspired by tools like **lazydocker** and **k9s** — but built for managing your fleet of servers directly from your terminal.

With **lazyssh**, you can quickly **navigate**, **connect**, **manage**, and **transfer files** between your local machine and any server defined in your `~/.ssh/config`.  
No more remembering IP addresses or running long `scp` commands — just a clean, keyboard-driven UI.

---

## ✨ Features

### Server Management 
- 📜 **Read & display** servers from your `~/.ssh/config` in a scrollable list.
- ➕ **Add** a new server entry from the UI by specifying:
  - Host alias
  - HostName / IP
  - Username
  - Port
  - Identity file
- ✏ **Edit** existing server entries directly from the UI.
- 🗑 **Delete** server entries safely.

### **Quick Server Navigation**
- 🔍 **Fuzzy search** through servers by alias or IP.
- ⏩ Instant SSH into selected server with a single keypress.
- 🏷 Grouping/tagging of servers (e.g., `prod`, `dev`, `test`) for quick filtering.

### **Remote Operations**
- 🖥 **Open Terminal**: Start an SSH session instantly.
- 📤 **Copy from server → local**: Select remote file/folder, choose local destination.
- 📥 **Copy from local → server**: Select local file/folder, choose remote destination.

### **Port Forwarding**
- 📡 Easily forward local ports to remote services (and vice versa) from the UI.
- Save & reuse common port forwarding setups.

---

## 🎯 Use Cases

- Developers switching between dozens of dev/test/staging/production VMs
- Sysadmins managing multiple environments and needing quick access
- Anyone who wants **fast, zero-hassle SSH management** without memorizing IPs

---

