#!/usr/bin/env bash
rm -rf bin/*
set -euo pipefail

# LazySSH Build Script
# Usage: ./build.sh [options]
#
# Options:
#   -o, --output DIR     Output directory (default: ./bin)
#   -v, --version VER    Set version string (default: from git tag or "develop")
#   -c, --commit HASH    Set git commit hash (default: auto-detect)
#   -a, --arch ARCH      Target architecture (default: current arch)
#   -s, --os OS          Target OS (default: current os)
#   -r, --race           Enable race detector (debug build)
#   -p, --parallel       Build for all platforms
#   -l, --lang LANG      Set default language at runtime
#                           Supported: en (default), zh-CN
#   -h, --help           Show this help message

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Default values
OUTPUT_DIR="./bin"
VERSION="develop"
COMMIT="unknown"
RACE=false
PARALLEL=false
LANG="en"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    head -16 "$0" | tail -12
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--output) OUTPUT_DIR="$2"; shift 2 ;;
        -v|--version) VERSION="$2"; shift 2 ;;
        -c|--commit) COMMIT="$2"; shift 2 ;;
        -a|--arch) BUILD_ARCH="$2"; shift 2 ;;
        -s|--os) BUILD_OS="$2"; shift 2 ;;
        -l|--lang)
            raw_lang="$2"
            # Normalize: lowercase for detection, then map to standard form
            lower=$(echo "$raw_lang" | tr '[:upper:]' '[:lower:]')
            case "$lower" in
                zh*|cn*|chs*|zh_cn|zh-cn|zh_cn|chinese)
                    LANG="zh-CN"
                    ;;
                en*|english)
                    LANG="en"
                    ;;
                *)
                    log_warn "Unknown language '$raw_lang', using English"
                    LANG="en"
                    ;;
            esac
            shift 2
            ;;
        -r|--race) RACE=true; shift ;;
        -p|--parallel) PARALLEL=true; shift ;;
        -h|--help) show_help ;;
        *) log_error "Unknown option: $1"; show_help ;;
    esac
done

# Auto-detect git commit if not provided
if [[ "$COMMIT" == "unknown" ]]; then
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
fi

# Build flags
LDFLAGS=(-ldflags "-X main.version=${VERSION} -X main.gitCommit=${COMMIT} -X github.com/Adembc/lazyssh/internal/i18n.defaultLang=${LANG}")

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Build function
build_binary() {
    local target_os="$1"
    local target_arch="$2"
    local output_name="$3"

    log_info "Building for ${target_os}/${target_arch}..."

    if [[ "$RACE" == true ]]; then
        log_warn "Race detector enabled (debug build)"
        GOOS=${target_os} GOARCH=${target_arch} go build -race "${LDFLAGS[@]}" -o "${OUTPUT_DIR}/${output_name}" ./cmd
    else
        GOOS=${target_os} GOARCH=${target_arch} go build "${LDFLAGS[@]}" -o "${OUTPUT_DIR}/${output_name}" ./cmd
    fi

    if [[ $? -eq 0 ]]; then
        log_success "Built: ${OUTPUT_DIR}/${output_name}"
    else
        log_error "Failed to build: ${output_name}"
        return 1
    fi
}

LANG_SUFFIX=""
if [[ "$LANG" != "en" ]]; then
    LANG_SUFFIX="-${LANG}"
fi

log_info "LazySSH Build"
log_info "Version:  ${VERSION}"
log_info "Commit:   ${COMMIT}"
log_info "Language: ${LANG}"
log_info "Output:   ${OUTPUT_DIR}"
echo ""

rm -f "${OUTPUT_DIR}"/lazyssh* 2>/dev/null || true

if [[ "$PARALLEL" == true ]]; then
    log_info "Building for all platforms..."
    echo ""

    build_binary "linux"   "amd64"   "lazyssh${LANG_SUFFIX}-linux-amd64"
    build_binary "linux"   "arm64"   "lazyssh${LANG_SUFFIX}-linux-arm64"
    build_binary "darwin"  "amd64"   "lazyssh${LANG_SUFFIX}-darwin-amd64"
    build_binary "darwin"  "arm64"   "lazyssh${LANG_SUFFIX}-darwin-arm64"
    build_binary "windows" "amd64"   "lazyssh${LANG_SUFFIX}-windows-amd64.exe"

    echo ""
    log_success "All builds complete!"
else
    CURRENT_OS=$(go env GOOS)
    CURRENT_ARCH=$(go env GOARCH)

    build_binary "$CURRENT_OS" "$CURRENT_ARCH" "lazyssh${LANG_SUFFIX}"

    echo ""
    log_success "Build complete: ${OUTPUT_DIR}/lazyssh${LANG_SUFFIX}"
    log_info "Run: ${OUTPUT_DIR}/lazyssh${LANG_SUFFIX}"
fi
