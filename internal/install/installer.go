package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform identifies the target AI coding assistant.
type Platform string

const (
	PlatformClaudeCode Platform = "claude"
	PlatformCursor     Platform = "cursor"
	PlatformCodex      Platform = "codex"
	PlatformAider      Platform = "aider"
)

// Options controls what Install writes.
type Options struct {
	Platform Platform
	// BinaryPath is the cortix binary to register in MCP configs.
	// Defaults to the current executable.
	BinaryPath string
	// ProjectDir is the working directory to write project-level configs into.
	// Empty means skip project-level files.
	ProjectDir string
}

// Install registers cortix as a skill and MCP server with the given AI assistant.
func Install(opts Options) error {
	if opts.BinaryPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("install: resolve binary path: %w", err)
		}
		opts.BinaryPath = exe
	}

	switch opts.Platform {
	case PlatformClaudeCode, "":
		return installClaude(opts)
	case PlatformCursor:
		return installCursor(opts)
	case PlatformCodex:
		return installCodex(opts)
	case PlatformAider:
		return installAider(opts)
	default:
		return fmt.Errorf("install: unknown platform %q — valid: claude, cursor, codex, aider", opts.Platform)
	}
}

// Uninstall removes cortix registrations from the given platform.
func Uninstall(platform Platform) error {
	switch platform {
	case PlatformClaudeCode, "":
		return uninstallClaude()
	case PlatformCursor:
		return uninstallCursor()
	default:
		return fmt.Errorf("uninstall: unknown platform %q — valid: claude, cursor", platform)
	}
}

// --- Claude Code ---

func installClaude(opts Options) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("install: home dir: %w", err)
	}

	// 1. Write ~/.claude/skills/cortix/SKILL.md
	skillDir := filepath.Join(home, ".claude", "skills", "cortix")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("install: create skill dir: %w", err)
	}
	if err := writeFile(filepath.Join(skillDir, "SKILL.md"), skillMD()); err != nil {
		return fmt.Errorf("install: write SKILL.md: %w", err)
	}
	fmt.Printf("  wrote  %s\n", filepath.Join(skillDir, "SKILL.md"))

	// 2. Patch ~/.claude/CLAUDE.md — add trigger line if not already present
	claudeMDPath := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := patchClaudeMD(claudeMDPath); err != nil {
		return fmt.Errorf("install: patch CLAUDE.md: %w", err)
	}
	fmt.Printf("  patched %s\n", claudeMDPath)

	// 3. Write ~/.claude/.mcp.json — add cortix server entry
	mcpPath := filepath.Join(home, ".claude", ".mcp.json")
	if err := patchMCPJSON(mcpPath, opts.BinaryPath); err != nil {
		return fmt.Errorf("install: patch .mcp.json: %w", err)
	}
	fmt.Printf("  patched %s\n", mcpPath)

	fmt.Println("\ncortix is ready. Open Claude Code and type /cortix to use it.")
	fmt.Println("cortix — github.com/JunaCodeBase/cortix")
	return nil
}

func uninstallClaude() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("uninstall: home dir: %w", err)
	}

	// Remove skill dir
	skillDir := filepath.Join(home, ".claude", "skills", "cortix")
	if err := os.RemoveAll(skillDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("uninstall: remove skill dir: %w", err)
	}
	fmt.Printf("  removed %s\n", skillDir)

	// Remove trigger block from CLAUDE.md
	claudeMDPath := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := unpatchClaudeMD(claudeMDPath); err != nil {
		return fmt.Errorf("uninstall: unpatch CLAUDE.md: %w", err)
	}
	fmt.Printf("  cleaned %s\n", claudeMDPath)

	// Remove cortix entry from .mcp.json
	mcpPath := filepath.Join(home, ".claude", ".mcp.json")
	if err := removeMCPEntry(mcpPath); err != nil {
		return fmt.Errorf("uninstall: remove MCP entry: %w", err)
	}
	fmt.Printf("  cleaned %s\n", mcpPath)

	fmt.Println("\ncortix uninstalled from Claude Code.")
	return nil
}

// --- Cursor ---

func installCursor(opts Options) error {
	if opts.ProjectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("install cursor: working dir: %w", err)
		}
		opts.ProjectDir = cwd
	}

	// .cursor/mcp.json in the project directory
	cursorDir := filepath.Join(opts.ProjectDir, ".cursor")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		return fmt.Errorf("install cursor: create .cursor dir: %w", err)
	}
	mcpPath := filepath.Join(cursorDir, "mcp.json")
	if err := patchMCPJSON(mcpPath, opts.BinaryPath); err != nil {
		return fmt.Errorf("install cursor: patch mcp.json: %w", err)
	}
	fmt.Printf("  patched %s\n", mcpPath)

	fmt.Println("\ncortix MCP server registered for Cursor.")
	fmt.Println("Reload Cursor — cortix_scan, cortix_deep_scan, cortix_export_preview are now available as tools.")
	return nil
}

func uninstallCursor() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("uninstall cursor: %w", err)
	}
	mcpPath := filepath.Join(cwd, ".cursor", "mcp.json")
	if err := removeMCPEntry(mcpPath); err != nil {
		return fmt.Errorf("uninstall cursor: %w", err)
	}
	fmt.Printf("  cleaned %s\n", mcpPath)
	fmt.Println("\ncortix uninstalled from Cursor.")
	return nil
}

// --- Codex ---

func installCodex(opts Options) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("install codex: %w", err)
	}
	// Codex reads AGENTS.md in the project root
	agentsPath := filepath.Join(opts.ProjectDir, "AGENTS.md")
	if opts.ProjectDir == "" {
		agentsPath = "AGENTS.md"
	}
	block := "\n## cortix\nUse `cortix scan` to get cluster health data before answering Kubernetes questions.\nUse `cortix scan deep` for security, reliability, cost, or operations analysis.\n"
	if err := patchTextFile(agentsPath, block, "## cortix"); err != nil {
		return fmt.Errorf("install codex: patch AGENTS.md: %w", err)
	}
	fmt.Printf("  patched %s\n", agentsPath)
	_ = home
	fmt.Println("\ncortix registered for Codex via AGENTS.md.")
	return nil
}

// --- Aider ---

func installAider(_ Options) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("install aider: %w", err)
	}
	// Aider reads ~/.aider.conf.yml — print instructions since YAML patching is brittle
	fmt.Printf("Add this to %s:\n\n", filepath.Join(home, ".aider.conf.yml"))
	fmt.Printf("  read: [%s]\n\n", skillMDPath(home))
	fmt.Println("Run 'cortix install' first to write the SKILL.md to that path.")
	return nil
}

// --- MCP JSON helpers ---

func patchMCPJSON(path, binaryPath string) error {
	data := map[string]any{}

	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	}

	servers, _ := data["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	servers["cortix"] = mcpServerEntry(binaryPath)
	data["mcpServers"] = servers

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, string(out)+"\n")
}

func removeMCPEntry(path string) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	data := map[string]any{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if servers, ok := data["mcpServers"].(map[string]any); ok {
		delete(servers, "cortix")
		if len(servers) == 0 {
			delete(data, "mcpServers")
		}
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, string(out)+"\n")
}

func mcpServerEntry(binaryPath string) map[string]any {
	// During development (go run), use the source path.
	// Once installed as a binary, just use the executable directly.
	if strings.HasSuffix(binaryPath, "cortix") || strings.HasSuffix(binaryPath, "cortix.exe") {
		return map[string]any{
			"command": binaryPath,
			"args":    []string{"mcp"},
		}
	}
	// Fallback: assume cortix is on PATH
	cmd := "cortix"
	if runtime.GOOS == "windows" {
		cmd = "cortix.exe"
	}
	return map[string]any{
		"command": cmd,
		"args":    []string{"mcp"},
	}
}

// --- CLAUDE.md helpers ---

const claudeMDBlock = `
# cortix
- **cortix** (` + "`~/.claude/skills/cortix/SKILL.md`" + `) - Kubernetes cluster intelligence via Cortix MCP tools. Trigger: ` + "`/cortix`" + `
When the user types ` + "`/cortix`" + `, invoke the Skill tool with ` + "`skill: \"cortix\"`" + ` before doing anything else.
The skill uses MCP tools cortix_scan, cortix_deep_scan, and cortix_export_preview to get live cluster data.
`

func patchClaudeMD(path string) error {
	existing := ""
	if raw, err := os.ReadFile(path); err == nil {
		existing = string(raw)
	}
	if strings.Contains(existing, "# cortix") {
		return nil // already installed
	}
	return writeFile(path, existing+claudeMDBlock)
}

func unpatchClaudeMD(path string) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// Remove everything from "# cortix\n" to the next blank-line-separated block
	content := string(raw)
	start := strings.Index(content, "\n# cortix\n")
	if start == -1 {
		return nil
	}
	// Find the end: next "\n# " heading or EOF
	rest := content[start+1:]
	end := strings.Index(rest[len("# cortix\n"):], "\n# ")
	if end == -1 {
		content = content[:start]
	} else {
		content = content[:start] + "\n" + rest[len("# cortix\n")+end+1:]
	}
	return writeFile(path, strings.TrimRight(content, "\n")+"\n")
}

func patchTextFile(path, block, sentinel string) error {
	existing := ""
	if raw, err := os.ReadFile(path); err == nil {
		existing = string(raw)
	}
	if strings.Contains(existing, sentinel) {
		return nil
	}
	return writeFile(path, existing+block)
}

// --- misc ---

func skillMDPath(home string) string {
	return filepath.Join(home, ".claude", "skills", "cortix", "SKILL.md")
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
