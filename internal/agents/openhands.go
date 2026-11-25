package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// openhands 向 OpenHands（原 OpenDevin）写入 LLM 配置，使其模型选择器
// 指向 zen-gate 本地网关。配置文件先备份，关闭即还原。
type openhands struct{}

func newOpenHands() *openhands { return &openhands{} }

func (o *openhands) Meta() (string, string, string) {
	return "openhands", "OpenHands", "~/.openhands/config.toml 的 [llm] 指向本地网关（原值自动备份还原）"
}

func (o *openhands) configDir() string { return homePath(".openhands") }
func (o *openhands) configPath() string {
	return filepath.Join(o.configDir(), "config.toml")
}

func (o *openhands) Detect() (bool, string, string) {
	if _, err := os.Stat(o.configDir()); err == nil {
		return true, "", "检测到 ~/.openhands"
	}
	return false, "", "未检测到 OpenHands"
}

func (o *openhands) IsEnabled() (bool, string, error) {
	data, err := os.ReadFile(o.configPath())
	if err != nil {
		return false, "", nil
	}
	return strings.Contains(string(data), "# >>> zen-gate"), "", nil
}

func (o *openhands) Enable(opts Options) error {
	path := o.configPath()
	if data, err := os.ReadFile(path); err == nil {
		_, _ = backupFile("openhands", path, data)
	}
	if err := os.MkdirAll(o.configDir(), 0o700); err != nil {
		return err
	}

	cfg := o.buildConfig(opts)
	return atomicWrite(path, cfg)
}

func (o *openhands) Disable() error {
	path := o.configPath()
	if b := latestBackup("openhands", "config.toml"); b != "" {
		data, err := os.ReadFile(b)
		if err == nil {
			return atomicWrite(path, data)
		}
	}
	// 无备份：仅删除 zen-gate 管理的 block（若有）
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	clean := stripZenGateBlock(string(data))
	return atomicWrite(path, []byte(clean))
}

// buildConfig 生成 OpenHands 的 config.toml 内容，包含 zen-gate 管理标记。
func (o *openhands) buildConfig(opts Options) string {
	var b strings.Builder
	b.WriteString("# >>> zen-gate managed block (do not edit)
")
	b.WriteString("[llm]
")
	b.WriteString(fmt.Sprintf("model = %q
", opts.DefaultModel))
	b.WriteString(fmt.Sprintf("api_key = %q
", opts.APIKey))
	b.WriteString(fmt.Sprintf("base_url = %q
", strings.TrimSuffix(opts.BaseURL, "/v1")))
	b.WriteString("# <<< zen-gate managed block
")
	return b.String()
}

// stripZenGateBlock 移除 zen-gate 管理的 [llm] block，保留用户原有内容。
func stripZenGateBlock(src string) string {
	lines := strings.Split(src, "
")
	out := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "# >>> zen-gate"):
			inBlock = true
			continue
		case strings.HasPrefix(t, "# <<< zen-gate"):
			inBlock = false
			continue
		case !inBlock:
			out = append(out, line)
		}
	}
	return strings.TrimRight(strings.Join(out, "
"), "
")
}

// 确保 openhands 实现了 Agent 接口。
var _ Agent = (*openhands)(nil)
