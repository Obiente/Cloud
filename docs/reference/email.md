# Email Template & SMTP

Transactional emails in Obiente Cloud share a single responsive template and are delivered through the built-in SMTP sender. This document explains how to configure SMTP, customise email content, and trigger messages from the API.

## SMTP Configuration Overview

- Configure the SMTP connection with the `SMTP_*` variables documented in [Environment Variables](environment-variables.md#email-delivery).
- Set `DASHBOARD_URL` so links inside emails point to the correct dashboard.
- Optionally set `SUPPORT_EMAIL` to surface a help contact in the footer.

When `SMTP_HOST` or `SMTP_FROM_ADDRESS` are missing, the API falls back to a no-op sender and logs a single reminder that email delivery is disabled.

The notifications service treats missing SMTP configuration as a retryable delivery failure. Notifications whose preferences permit email remain queued until configuration is restored.

## Notification delivery recovery

The notifications service commits each new notification and its email delivery job in one database transaction. Workers claim jobs from `notification_email_deliveries`, so bursts, service restarts and SMTP outages do not discard accepted work. Existing notifications are not retroactively queued during upgrade.

Four workers poll every five seconds and retry failures with backoff from 30 seconds to 15 minutes. Each send has a 30-second timeout; abandoned claims become eligible again after a two-minute lease. SMTP connections also respect cancellation and a deadline for the entire exchange, including the server greeting. Multiple service replicas can safely share the outbox.

Delivery is at least once. If SMTP accepts a message but the database acknowledgement fails, recovery may send a duplicate. Current email preferences are checked before each attempt; disabled email or a severity below the user's threshold completes the job without sending.

Operators can inspect aggregate queue state without exposing notification content:

```sql
SELECT status, count(*), min(created_at) AS oldest
FROM notification_email_deliveries
GROUP BY status;
```

`pending` includes retries, `processing` has an active or recoverable lease, `completed` includes preference-based skips, and `failed` retains an invalid payload for investigation. Review `available_at`, `lease_until`, `attempts` and `last_error` when delivery stalls. Payloads and errors can contain private data; restrict access. Completed rows remain available for retention policies, and pending or processing rows must not be removed by cleanup jobs.

## Rendering Email Content

All transactional messages share the template defined in `internal/email/templates/base.html`. Populate it by constructing a `TemplateData` payload:

```go
tmpl := email.TemplateData{
	Subject:     "Welcome to Obiente Cloud",
	PreviewText: "Get started with your new organization.",
	Greeting:    "Hi Ada,",
	Heading:     "You're all set",
	IntroLines: []string{
		"Thanks for choosing Obiente Cloud.",
		"Here are a few quick links to explore.",
	},
	Highlights: []email.Highlight{{Label: "Plan", Value: "Starter"}},
	Sections: []email.Section{
		{
			Title: "Next steps",
			Lines: []string{"Invite a teammate", "Connect your GitHub repository"},
		},
	},
	CTA: &email.CTA{Label: "Open console", URL: consoleURL},
	Category:     email.CategoryOnboarding,
	SupportEmail: supportEmail,
}

html, _ := email.RenderHTML(tmpl)
text := email.RenderText(tmpl)
```

The renderer automatically fills signature, footer, and colour accents based on the selected `Category` (`onboarding`, `invite`, `notification`, `billing`, or `system`).

## Sending Emails

Use `email.NewSenderFromEnv()` to create a sender configured from environment variables. Compose a `Message` and call `Send`:

```go
sender := email.NewSenderFromEnv()
if sender.Enabled() {
	msg := &email.Message{
		To:       []string{"team@example.com"},
		Template: &tmpl,
		Category: email.CategoryNotification,
		Metadata: map[string]string{"event": "welcome"},
	}
	if err := sender.Send(ctx, msg); err != nil {
		log.Printf("email send failed: %v", err)
	}
}
```

- `Template` can be omitted when you provide `HTMLBody` / `TextBody` manually.
- `Metadata` entries are promoted to headers prefixed with `X-Obiente-` for downstream processing.
- If you'd like to override the default button colour, set `AccentColor` on `TemplateData`.

## Invitation Emails

Organization invitations now trigger automatically from `InviteMember`. The service:

- Builds the invite email with organization, role, and CTA details.
- Sends via the configured SMTP sender.
- Falls back gracefully (with logs) when SMTP is not configured.

To customise invite wording further, adjust `dispatchInviteEmail` in `internal/services/organizations/service.go`.

## Troubleshooting

1. **No emails arrive** – confirm `SMTP_HOST` and `SMTP_FROM_ADDRESS` are set and the API logs “SMTP enabled”.
2. **TLS errors** – verify the SMTP host supports STARTTLS on the configured port; use `SMTP_SKIP_TLS_VERIFY=true` only in controlled environments.
3. **Broken links** – set `DASHBOARD_URL` so the CTA points to the correct dashboard host.

---

[← Back to Reference](index.md)

