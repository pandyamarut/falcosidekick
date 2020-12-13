package outputs

import (
	"bytes"
	"log"
	"strings"

	"github.com/falcosecurity/falcosidekick/types"
)

// Field
type slackAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Attachment
type slackAttachment struct {
	Fallback   string                 `json:"fallback"`
	Color      string                 `json:"color"`
	Text       string                 `json:"text,omitempty"`
	Fields     []slackAttachmentField `json:"fields"`
	Footer     string                 `json:"footer,omitempty"`
	FooterIcon string                 `json:"footer_icon,omitempty"`
}

// Payload
type slackPayload struct {
	Text        string            `json:"text,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

func newSlackPayload(falcopayload types.FalcoPayload, config *types.Configuration) slackPayload {
	var (
		messageText string
		attachments []slackAttachment
		attachment  slackAttachment
		fields      []slackAttachmentField
		field       slackAttachmentField
	)

	if config.Slack.OutputFormat == All || config.Slack.OutputFormat == Fields || config.Slack.OutputFormat == "" {
		for i, j := range falcopayload.OutputFields {
			switch v := j.(type) {
			case string:
				field.Title = i
				field.Value = v
				if len([]rune(v)) < 36 {
					field.Short = true
				} else {
					field.Short = false
				}
			default:
				continue
			}

			fields = append(fields, field)
		}

		field.Title = Rule
		field.Value = falcopayload.Rule
		field.Short = true
		fields = append(fields, field)
		field.Title = Priority
		field.Value = falcopayload.Priority
		field.Short = true
		fields = append(fields, field)
		field.Title = Time
		field.Short = false
		field.Value = falcopayload.Time.String()
		fields = append(fields, field)

		attachment.Footer = DefaultFooter
		if config.Slack.Footer != "" {
			attachment.Footer = config.Slack.Footer
		}
	}

	attachment.Fallback = falcopayload.Output
	attachment.Fields = fields
	if config.Slack.OutputFormat == All || config.Slack.OutputFormat == Fields || config.Slack.OutputFormat == "" {
		attachment.Text = falcopayload.Output
	}

	if config.Slack.MessageFormatTemplate != nil {
		buf := &bytes.Buffer{}
		if err := config.Slack.MessageFormatTemplate.Execute(buf, falcopayload); err != nil {
			log.Printf("[ERROR] : Slack - Error expanding Slack message %v", err)
		} else {
			messageText = buf.String()
		}
	}

	var color string
	switch strings.ToLower(falcopayload.Priority) {
	case Emergency:
		color = Red
	case Alert:
		color = Orange
	case Critical:
		color = Orange
	case Error:
		color = Red
	case Warning:
		color = Yellow
	case Notice:
		color = Lightcyan
	case Informational:
		color = LigthBlue
	case Debug:
		color = PaleCyan
	}
	attachment.Color = color

	attachments = append(attachments, attachment)

	s := slackPayload{
		Text:        messageText,
		Username:    config.Slack.Username,
		IconURL:     config.Slack.Icon,
		Attachments: attachments}

	return s
}

// SlackPost posts event to Slack
func (c *Client) SlackPost(falcopayload types.FalcoPayload) {
	c.Stats.Slack.Add(Total, 1)

	err := c.Post(newSlackPayload(falcopayload, c.Config))
	if err != nil {
		go c.CountMetric(Outputs, 1, []string{"output:slack", "status:error"})
		c.Stats.Slack.Add(Error, 1)
		c.PromStats.Outputs.With(map[string]string{"destination": "slack", "status": Error}).Inc()
		log.Printf("[ERROR] : Slack - %v\n", err)
		return
	}

	// Setting the success status
	go c.CountMetric(Outputs, 1, []string{"output:slack", "status:ok"})
	c.Stats.Slack.Add(OK, 1)
	c.PromStats.Outputs.With(map[string]string{"destination": "slack", "status": OK}).Inc()
	log.Printf("[INFO] : Slack - Publish OK\n")
}
