package stateful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/agence-webup/backr/manager"
)

/*
{
    "attachments": [
        {
            "fallback": "Required plain-text summary of the attachment.",
            "color": "warning",
            "title": "CPU/Load issue",
            "fields": [
                {
                    "title": "Server",
                    "value": "q-demos2",
                    "short": true
                },
				{
                    "title": "IP",
                    "value": "23.53.154.12",
                    "short": true
                },
				{
                    "title": "Level",
                    "value": "Warning",
                    "short": true
                }
            ],
            "ts": 123456789
        }
    ]
}
*/

type slackPayload struct {
	Attachments []slackPayloadAttachment `json:"attachments"`
}

type slackPayloadAttachment struct {
	Fallback string                        `json:"fallback"`
	Color    string                        `json:"color"`
	Title    string                        `json:"title"`
	Fields   []slackPayloadAttachmentField `json:"fields"`
}

type slackPayloadAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func sendSlackMessage(webhookURL string, notif notification) error {

	// check if a webhook URL is set
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is not set")
	}

	// prepare payload
	payload := getPayload(notif)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cannot marshal payload into json: %w", err)
	}

	// prepare the request
	b := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", webhookURL, b)
	if err != nil {
		return fmt.Errorf("cannot prepare request: %w", err)
	}

	client := http.Client{
		Timeout: time.Duration(5) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error with request to slack API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("cannot read response body: %w", err)
		}
		return fmt.Errorf("error with Slack webhook: %v", string(body))
	}

	return nil
}

func getPayload(notif notification) slackPayload {

	reason := ""
	for r, desc := range notif.Statement.Reasons {
		reason += fmt.Sprintf(" - %v: %v\n", r.String(), desc)
	}

	return slackPayload{
		Attachments: []slackPayloadAttachment{
			slackPayloadAttachment{
				Title:    "Backup issue",
				Color:    getSlackColorForLevel(&notif.Statement.MaxLevel),
				Fallback: fmt.Sprintf("%v: %s on '%v'", notif.Statement.MaxLevel, "Backup issue", notif.Statement.Project.Name),
				Fields: []slackPayloadAttachmentField{
					slackPayloadAttachmentField{
						Title: "Project",
						Value: notif.Statement.Project.Name,
						Short: true,
					},
					slackPayloadAttachmentField{
						Title: "Created at",
						Value: notif.CreatedAt.UTC().Format(time.RFC822),
						Short: true,
					},
					slackPayloadAttachmentField{
						Title: "Reasons",
						Value: reason,
						Short: false,
					},
				},
			},
		},
	}
}

func getSlackColorForLevel(level *manager.AlertLevel) string {
	if level != nil {
		switch *level {
		case manager.Critic:
			return "danger"
		case manager.Warning:
			return "warning"
		}
	}
	return "good"
}
