package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// openhands 向 OpenHands Agent Canvas 写入 LLM 配置，使其模型选择器
// 指向 zen-gate 本地网关。配置文件先备份，关闭即还原。
type openhands struct{}

func newOpenHands() *openhands { return &openhands{} }

func (o *openhands) Meta() (string, string, string) {
	return "openhands", "OpenHands", "~/.openhands/settings.json 的 agent_settings.llm 指向本地网关（原值自动备份还原）"
}

func (o *openhands) configDir() string { return homePath(".openhands") }
func (o *openhands) configPath() string {
	return filepath.Join(o.configDir(), "settings.json")
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
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return false, "", nil
	}
	agentSettings, ok := settings["agent_settings"].(map[string]interface{})
	if !ok {
		return false, "", nil
	}
	managed, _ := agentSettings["zen_gate_managed"].(bool)
	return managed, "", nil
}

func (o *openhands) Enable(opts Options) error {
	path := o.configPath()
	if data, err := os.ReadFile(path); err == nil {
		_, _ = backupFile("openhands", path, data)
	}
	if err := os.MkdirAll(o.configDir(), 0o700); err != nil {
		return err
	}

	// 读取现有 settings.json
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	// 确保 agent_settings 存在
	agentSettings, ok := settings["agent_settings"].(map[string]interface{})
	if !ok {
		agentSettings = make(map[string]interface{})
		settings["agent_settings"] = agentSettings
	}

	// 确保 llm 存在
	llm, ok := agentSettings["llm"].(map[string]interface{})
	if !ok {
		llm = make(map[string]interface{})
		agentSettings["llm"] = llm
	}

	// 设置 zen-gate 配置
	llm["model"] = opts.DefaultModel
	llm["api_key"] = opts.APIKey
	llm["base_url"] = opts.BaseURL
	// 标记 zen-gate 管理
	agentSettings["zen_gate_managed"] = true

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, out)
}

func (o *openhands) Disable() error {
	path := o.configPath()
	if b := latestBackup("openhands", "settings.json"); b != "" {
		data, err := os.ReadFile(b)
		if err == nil {
			return atomicWrite(path, data)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}
	agentSettings, ok := settings["agent_settings"].(map[string]interface{})
	if ok {
		delete(agentSettings, "zen_gate_managed")
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil
	}
	return atomicWrite(path, out)
}

var _ Agent = (*openhands)(nil)
