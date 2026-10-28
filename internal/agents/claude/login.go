package claude

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
	"github.com/Glacius-Labs/Markitect/internal/agents"
)

// apiKeyPrefix starts every Anthropic API key; an account login never
// carries one.
const apiKeyPrefix = "sk-ant-api"

// Login admits a Claude Code credentials file (.credentials.json) with a
// claude.ai account login. The Manager's copy keeps only claudeAiOauth
// without its refresh token, so the Manager can neither refresh nor rotate
// the login; other entries, such as MCP server logins, never reach it. An
// access token past expiresAt is refused: refresh the login on the host by
// running claude there. There is no API-key route and no long-lived token.
func (Provider) Login(source []byte, now time.Time) (agents.Login, error) {
	var file map[string]json.RawMessage
	if len(source) == 0 || len(source) > frame.MaxPrivateAuthBytes || json.Unmarshal(source, &file) != nil {
		return agents.Login{}, errors.New("claude: the login is not a Claude Code credentials file")
	}
	var oauth map[string]json.RawMessage
	if json.Unmarshal(file["claudeAiOauth"], &oauth) != nil || oauth == nil {
		return agents.Login{}, errors.New("claude: the credentials file holds no claudeAiOauth account login")
	}
	var access, refresh string
	if json.Unmarshal(oauth["accessToken"], &access) != nil || access == "" {
		return agents.Login{}, errors.New("claude: the account login has no access token")
	}
	if strings.HasPrefix(access, apiKeyPrefix) {
		return agents.Login{}, errors.New("claude: the login holds an Anthropic API key; only an account login is admitted")
	}
	login := agents.Login{Secrets: [][]byte{[]byte(access)}}
	if raw, ok := oauth["refreshToken"]; ok && string(raw) != "null" {
		if json.Unmarshal(raw, &refresh) != nil {
			return agents.Login{}, errors.New("claude: the account login has a malformed refresh token")
		}
		if refresh != "" {
			login.Secrets = append(login.Secrets, []byte(refresh))
		}
	}
	if raw, ok := oauth["expiresAt"]; ok && string(raw) != "null" {
		var ms int64
		if json.Unmarshal(raw, &ms) != nil {
			return agents.Login{}, errors.New("claude: the account login has a malformed expiry")
		}
		login.Expires = time.UnixMilli(ms)
		if !now.Before(login.Expires) {
			return agents.Login{}, errors.New("claude: the login's access token has expired; run claude on the host to refresh it")
		}
	}
	delete(oauth, "refreshToken")
	copied, err := json.Marshal(map[string]map[string]json.RawMessage{"claudeAiOauth": oauth})
	if err != nil {
		return agents.Login{}, err
	}
	login.Copy = copied
	return login, nil
}
