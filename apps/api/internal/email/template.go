package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

var baseTemplate = template.Must(template.New("base.html").Funcs(template.FuncMap{
	"join": strings.Join,
}).ParseFS(templateFS, "templates/base.html"))

// Category represents the high level type of email being sent.
type Category string

const (
	CategoryOnboarding   Category = "onboarding"
	CategoryInvite       Category = "invite"
	CategoryNotification Category = "notification"
	CategoryBilling      Category = "billing"
	CategorySystem       Category = "system"
)

// CTA represents a primary call-to-action button within the email.
type CTA struct {
	Label       string
	URL         string
	Description string
}

// Highlight allows showing key value pairs near the top of the email (e.g. plan, amount).
type Highlight struct {
	Label string
	Value string
}

// Bullet represents a single bullet item within a section.
type Bullet struct {
	Label       string
	Description string
}

// Section is a logical group of rich text content that can include bullets.
type Section struct {
	Title   string
	Lines   []string
	Bullets []Bullet
}

// TemplateData describes the dynamic data used to render the email template.
type TemplateData struct {
	Subject        string
	PreviewText    string
	Greeting       string
	Heading        string
	IntroLines     []string
	Highlights     []Highlight
	Sections       []Section
	CTA            *CTA
	SignatureLines []string
	SupportLines   []string
	FooterLines    []string
	Category       Category
	AccentColor    string
	BrandName      string
	BrandURL       string
	BaseURL        string
	SupportEmail   string
	Year           int
}

type viewModel struct {
	Subject        string
	PreviewText    string
	Greeting       string
	Heading        string
	IntroLines     []string
	Highlights     []Highlight
	Sections       []Section
	CTA            *CTA
	SignatureLines []string
	SupportLines   []string
	FooterLines    []string
	AccentColor    string
	CategoryLabel  string
	BrandName      string
	BrandURL       string
	BaseURL        string
	Year           int
}

var categoryDefaults = map[Category]struct {
	Accent string
	Label  string
}{
	CategoryOnboarding:   {Accent: "#2563eb", Label: "Onboarding"},
	CategoryInvite:       {Accent: "#0ea5e9", Label: "Invitation"},
	CategoryNotification: {Accent: "#6366f1", Label: "Notification"},
	CategoryBilling:      {Accent: "#f97316", Label: "Billing"},
	CategorySystem:       {Accent: "#10b981", Label: "System"},
}

// RenderHTML renders the base HTML email template using the provided data.
func RenderHTML(data TemplateData) (string, error) {
	vm := buildViewModel(data)

	var buf bytes.Buffer
	if err := baseTemplate.Execute(&buf, vm); err != nil {
		return "", fmt.Errorf("render html template: %w", err)
	}
	return buf.String(), nil
}

// RenderText produces a plain-text representation of the email for clients that do not support HTML.
func RenderText(data TemplateData) string {
	vm := buildViewModel(data)
	var b strings.Builder

	if vm.Subject != "" {
		b.WriteString(vm.Subject)
		b.WriteString("\n\n")
	}
	if vm.Greeting != "" {
		b.WriteString(vm.Greeting)
		b.WriteString("\n\n")
	}
	if vm.Heading != "" {
		b.WriteString(vm.Heading)
		b.WriteString("\n\n")
	}
	for _, line := range vm.IntroLines {
		if line == "" {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n\n")
	}
	if len(vm.Highlights) > 0 {
		for _, h := range vm.Highlights {
			if h.Label == "" && h.Value == "" {
				continue
			}
			if h.Label != "" {
				b.WriteString(h.Label)
				b.WriteString(": ")
			}
			b.WriteString(h.Value)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	for _, section := range vm.Sections {
		if section.Title != "" {
			b.WriteString(section.Title)
			b.WriteString("\n")
			b.WriteString(strings.Repeat("-", len(section.Title)))
			b.WriteString("\n")
		}
		for _, line := range section.Lines {
			if line == "" {
				continue
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		if len(section.Bullets) > 0 {
			for _, bullet := range section.Bullets {
				b.WriteString("- ")
				if bullet.Label != "" {
					b.WriteString(bullet.Label)
					if bullet.Description != "" {
						b.WriteString(" - ")
					} else {
						b.WriteString("\n")
						continue
					}
				}
				b.WriteString(bullet.Description)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	if vm.CTA != nil {
		if vm.CTA.Label != "" {
			b.WriteString(vm.CTA.Label)
			b.WriteString(": ")
		}
		b.WriteString(vm.CTA.URL)
		b.WriteString("\n")
		if vm.CTA.Description != "" {
			b.WriteString(vm.CTA.Description)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	for _, line := range vm.SignatureLines {
		if line == "" {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(vm.SupportLines) > 0 {
		b.WriteString("\n")
		for _, line := range vm.SupportLines {
			if line == "" {
				continue
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if len(vm.FooterLines) > 0 {
		b.WriteString("\n")
		for _, line := range vm.FooterLines {
			if line == "" {
				continue
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func buildViewModel(data TemplateData) viewModel {
	defaults := categoryDefaults[data.Category]

	accent := data.AccentColor
	if accent == "" {
		accent = defaults.Accent
	}
	if accent == "" {
		accent = "#2563eb"
	}

	categoryLabel := defaults.Label
	if categoryLabel == "" {
		categoryLabel = "Update"
	}

	year := data.Year
	if year == 0 {
		year = time.Now().UTC().Year()
	}

	brandName := data.BrandName
	if brandName == "" {
		brandName = "Obiente Cloud"
	}

	signature := data.SignatureLines
	if len(signature) == 0 {
		signature = []string{"Thanks,", brandName + " Team"}
	}

	support := data.SupportLines
	if len(support) == 0 && data.SupportEmail != "" {
		support = []string{"Need help? Email " + data.SupportEmail + "."}
	}

	footer := data.FooterLines
	if len(footer) == 0 {
		footer = []string{fmt.Sprintf("© %d %s. All rights reserved.", year, brandName)}
	}

	return viewModel{
		Subject:        data.Subject,
		PreviewText:    data.PreviewText,
		Greeting:       data.Greeting,
		Heading:        data.Heading,
		IntroLines:     append([]string{}, data.IntroLines...),
		Highlights:     append([]Highlight(nil), data.Highlights...),
		Sections:       append([]Section(nil), data.Sections...),
		CTA:            data.CTA,
		SignatureLines: append([]string{}, signature...),
		SupportLines:   append([]string{}, support...),
		FooterLines:    append([]string{}, footer...),
		AccentColor:    accent,
		CategoryLabel:  categoryLabel,
		BrandName:      brandName,
		BrandURL:       data.BrandURL,
		BaseURL:        data.BaseURL,
		Year:           year,
	}
}
