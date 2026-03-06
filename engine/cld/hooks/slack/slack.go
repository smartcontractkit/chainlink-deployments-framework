// Package slack provides reference hook implementations that post
// notifications to Slack via the Slack Bot API using Block Kit formatting
// inside colored attachments.
//
// The bot token is read from the SLACK_BOT_TOKEN environment variable at
// execution time. All hooks use [changeset.Warn] failure policy so Slack
// errors never block the changeset pipeline. When the token is empty the
// hooks log a warning and no-op, making them safe to register unconditionally.
//
// Usage:
//
//	Configure(myCS).With(cfg).
//	    WithPreHooks(slack.Notify(channel, "deploying tokens")).
//	    WithPostHooks(slack.Result(channel))
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

const (
	hookTimeout = 10 * time.Second

	// TokenEnvVar is the environment variable name read at hook execution time.
	TokenEnvVar = "SLACK_BOT_TOKEN" //nolint:gosec // env var name, not a credential

	colorBlue  = "#1976D2"
	colorGreen = "#2eb67d"
	colorRed   = "#e01e5a"
)

//nolint:gochecknoglobals // overridden in tests
var slackAPIPostURL = "https://slack.com/api/chat.postMessage"

// Block Kit + attachment types for structured Slack messages.

type textObj struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

type blockObj struct {
	Type     string    `json:"type"`
	Text     *textObj  `json:"text,omitempty"`
	Fields   []textObj `json:"fields,omitempty"`
	Elements []textObj `json:"elements,omitempty"`
}

type attachment struct {
	Color  string     `json:"color"`
	Blocks []blockObj `json:"blocks"`
}

type chatPostMessagePayload struct {
	Channel     string       `json:"channel"`
	Text        string       `json:"text"`
	Blocks      []blockObj   `json:"blocks,omitempty"`
	Attachments []attachment `json:"attachments"`
}

type chatPostMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func newHeader(text string) blockObj {
	return blockObj{
		Type: "header",
		Text: &textObj{Type: "plain_text", Text: text, Emoji: true},
	}
}

func newSection(text string) blockObj {
	return blockObj{
		Type: "section",
		Text: &textObj{Type: "mrkdwn", Text: text},
	}
}

func newFields(fields ...textObj) blockObj {
	return blockObj{Type: "section", Fields: fields}
}

func newContext(text string) blockObj {
	return blockObj{
		Type:     "context",
		Elements: []textObj{{Type: "mrkdwn", Text: text}},
	}
}

func mrkdwn(label, value string) textObj {
	return textObj{Type: "mrkdwn", Text: fmt.Sprintf("*%s*\n%s", label, value)}
}

// Notify returns a PreHook that posts a Block Kit message to channel using the
// Slack chat.postMessage API. The bot token is read from SLACK_BOT_TOKEN at
// execution time. Messages render with a blue color bar and structured fields.
// Uses Warn policy with a 10s timeout. No-ops when the env var is empty.
func Notify(channel, message string) changeset.PreHook {
	return changeset.PreHook{
		HookDefinition: changeset.HookDefinition{
			Name:          "slack-notify",
			FailurePolicy: changeset.Warn,
			Timeout:       hookTimeout,
		},
		Func: func(ctx context.Context, params changeset.PreHookParams) error {
			token := os.Getenv(TokenEnvVar)
			if token == "" {
				params.Env.Logger.Warnw("slack hook skipped: token is empty", "hook", "slack-notify", "env", TokenEnvVar)
				return nil
			}

			fallback := fmt.Sprintf("Changeset %s: %s", params.ChangesetKey, message)
			header := newHeader(":rocket: Changeset Starting")
			attBlocks := []blockObj{
				newFields(
					mrkdwn("Changeset", "`"+params.ChangesetKey+"`"),
					mrkdwn("Message", message),
				),
				newContext("Posted by deployment hooks"),
			}

			return post(ctx, token, channel, fallback, colorBlue, header, attBlocks)
		},
	}
}

// Result returns a PostHook that posts changeset success/failure status to
// channel using the Slack chat.postMessage API with Block Kit formatting.
// The bot token is read from SLACK_BOT_TOKEN at execution time.
// Success renders with a green color bar, failure with red.
// Uses Warn policy with a 10s timeout. No-ops when the env var is empty.
func Result(channel string) changeset.PostHook {
	return changeset.PostHook{
		HookDefinition: changeset.HookDefinition{
			Name:          "slack-result",
			FailurePolicy: changeset.Warn,
			Timeout:       hookTimeout,
		},
		Func: func(ctx context.Context, params changeset.PostHookParams) error {
			token := os.Getenv(TokenEnvVar)
			if token == "" {
				params.Env.Logger.Warnw("slack hook skipped: token is empty", "hook", "slack-result", "env", TokenEnvVar)
				return nil
			}

			var fallback, color string
			var header blockObj
			var attBlocks []blockObj

			if params.Err != nil {
				fallback = fmt.Sprintf("Changeset %s failed: %v", params.ChangesetKey, params.Err)
				color = colorRed
				header = newHeader(":x: Changeset Failed")
				attBlocks = []blockObj{
					newFields(
						mrkdwn("Changeset", "`"+params.ChangesetKey+"`"),
						mrkdwn("Status", ":x: Failed"),
					),
					newSection(fmt.Sprintf("*Error*\n> %v", params.Err)),
					newContext("Posted by deployment hooks"),
				}
			} else {
				fallback = fmt.Sprintf("Changeset %s succeeded", params.ChangesetKey)
				color = colorGreen
				header = newHeader(":white_check_mark: Changeset Succeeded")
				attBlocks = []blockObj{
					newFields(
						mrkdwn("Changeset", "`"+params.ChangesetKey+"`"),
						mrkdwn("Status", ":white_check_mark: Succeeded"),
					),
					newContext("Posted by deployment hooks"),
				}
			}

			return post(ctx, token, channel, fallback, color, header, attBlocks)
		},
	}
}

func post(ctx context.Context, token, channel, fallback, color string, header blockObj, attBlocks []blockObj) error {
	body, err := json.Marshal(chatPostMessagePayload{
		Channel: channel,
		Text:    fallback,
		Blocks:  []blockObj{header},
		Attachments: []attachment{{
			Color:  color,
			Blocks: attBlocks,
		}},
	})
	if err != nil {
		return fmt.Errorf("slack: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackAPIPostURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack: unexpected status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("slack: read response: %w", err)
	}

	var apiResp chatPostMessageResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("slack: unmarshal response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("slack: API error: %s", apiResp.Error)
	}

	return nil
}
