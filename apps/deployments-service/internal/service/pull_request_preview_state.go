package deployments

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

const pullRequestPreviewPath = "/previews/"
const pullRequestPreviewHostSuffix = ".my.obiente.cloud"

func pullRequestDeploymentStateURL(record *database.PullRequestDeployment) *string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PREVIEW_STATUS_BASE_URL")), "/")
	if base == "" {
		domain := strings.Trim(strings.TrimSpace(os.Getenv("DOMAIN")), ".")
		if domain != "" {
			base = "https://" + domain
		}
	}
	if base == "" || record == nil || record.ID == "" {
		return nil
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil
	}
	stateURL := strings.TrimRight(parsed.String(), "/") + pullRequestPreviewPath + url.PathEscape(record.ID)
	return &stateURL
}

func pullRequestDeploymentPublicURL(record *database.PullRequestDeployment) string {
	if record != nil && record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING) && record.EnvironmentURL != nil {
		return *record.EnvironmentURL
	}
	if stateURL := pullRequestDeploymentStateURL(record); stateURL != nil {
		return *stateURL
	}
	return ""
}

// HandlePullRequestPreviewState serves a public, deliberately metadata-light
// state page. The opaque record ID is the capability URL; organization,
// repository, log, error, and approver data are never rendered here.
func (s *Service) HandlePullRequestPreviewState(w http.ResponseWriter, r *http.Request) {
	setPreviewStateHeaders(w)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, pullRequestPreviewPath)
	if id == r.URL.Path || strings.Contains(id, "/") || !strings.HasPrefix(id, "pr-deployment-") || len(id) > 80 {
		http.NotFound(w, r)
		return
	}
	var record database.PullRequestDeployment
	if err := database.DB.WithContext(r.Context()).Where("id = ?", id).First(&record).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	renderPullRequestPreviewState(w, r, &record, true)
}

// HandlePullRequestPreviewHostFallback serves the state page directly on a PR
// hostname when Traefik has no healthy preview backend. It returns false for
// unrelated hosts so the normal deployments-service root handler can continue.
func (s *Service) HandlePullRequestPreviewHostFallback(w http.ResponseWriter, r *http.Request) bool {
	host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(r.Host), "."))
	if colon := strings.LastIndex(host, ":"); colon > -1 {
		host = host[:colon]
	}
	label := strings.TrimSuffix(host, pullRequestPreviewHostSuffix)
	if !strings.HasSuffix(host, pullRequestPreviewHostSuffix) || !strings.HasPrefix(label, "pr-") || !isDNSLabel(label) {
		return false
	}
	var record database.PullRequestDeployment
	httpsURL, httpURL := "https://"+host, "http://"+host
	if err := database.DB.WithContext(r.Context()).
		Where("environment_url IN ? OR environment_url LIKE ? OR environment_url LIKE ?", []string{httpsURL, httpURL}, httpsURL+"/%", httpURL+"/%").
		Order("updated_at DESC").
		First(&record).Error; err != nil {
		return false
	}
	setPreviewStateHeaders(w)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return true
	}
	renderPullRequestPreviewState(w, r, &record, false)
	return true
}

func renderPullRequestPreviewState(w http.ResponseWriter, r *http.Request, record *database.PullRequestDeployment, redirectRunning bool) {
	if redirectRunning && record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING) && record.EnvironmentURL != nil {
		http.Redirect(w, r, *record.EnvironmentURL, http.StatusTemporaryRedirect)
		return
	}
	title, detail, transient := previewStateCopy(record)
	if !redirectRunning && record.Status == int32(deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_RUNNING) {
		title = "Preview temporarily unavailable"
		detail = "The preview container is restarting or cannot be reached. This page will retry automatically."
		transient = true
	}
	if transient {
		w.Header().Set("Retry-After", "10")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	refresh := ""
	if transient {
		refresh = `<meta http-equiv="refresh" content="8">`
	}
	expiry := ""
	if record.ClosedAt == nil && !record.ExpiresAt.IsZero() {
		expiry = `<p class="expiry">This temporary preview shuts down automatically.</p>`
	}
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">%s<title>%s · Obiente Preview</title>
<style>:root{color-scheme:light dark;--bg:#f5f3f7;--panel:#fff;--text:#211d27;--muted:#6d6673;--line:#d9d3de;--accent:#7650a8}@media(prefers-color-scheme:dark){:root{--bg:#151219;--panel:#1d1922;--text:#f4eff8;--muted:#aaa1b1;--line:#37313d;--accent:#b795df}}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:24px;background:var(--bg);color:var(--text);font:16px/1.55 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.panel{width:min(560px,100%%);padding:32px;border:1px solid var(--line);border-radius:14px;background:var(--panel);box-shadow:0 12px 36px rgba(20,15,24,.08)}.mark{display:flex;align-items:center;gap:10px;margin-bottom:32px;font-size:14px;font-weight:700}.dot{width:10px;height:10px;border-radius:50%%;background:var(--accent)}h1{margin:0 0 10px;font-size:clamp(28px,6vw,42px);line-height:1.08;letter-spacing:-.03em}p{margin:0;color:var(--muted)}.expiry{margin-top:24px;padding-top:20px;border-top:1px solid var(--line);font-size:14px}</style></head>
<body><main class="panel"><div class="mark"><span class="dot"></span>Obiente Preview</div><h1>%s</h1><p>%s</p>%s</main></body></html>`, refresh, html.EscapeString(title), html.EscapeString(title), html.EscapeString(detail), expiry)
}

func setPreviewStateHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
}

func previewStateCopy(record *database.PullRequestDeployment) (title, detail string, transient bool) {
	switch deploymentsv1.PullRequestDeploymentStatus(record.Status) {
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_QUEUED:
		return "Queued", "The preview is waiting for a build slot. This page will update when it starts.", true
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_BUILDING:
		return "Building the preview", "The new revision is being built and started. This page will open it when it is ready.", true
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL:
		return "Waiting for approval", "A maintainer must approve this revision before any pull request code can run.", true
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_REJECTED:
		return "Preview not approved", "A maintainer chose not to run this revision.", false
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_FAILED:
		return "Preview unavailable", "The build did not finish successfully. Maintainers can inspect the build in Obiente.", false
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_SKIPPED:
		return "No preview for this change", "This pull request is outside the configured preview scope.", false
	case deploymentsv1.PullRequestDeploymentStatus_PULL_REQUEST_DEPLOYMENT_CLOSED:
		if record.Merged && record.ApprovedAt != nil {
			return "Preview is offline", "This merged pull request was previously approved. A maintainer can temporarily restore it from Obiente.", false
		}
		return "Preview is offline", "The temporary environment has been removed.", false
	default:
		return "Preview pending", "The preview state is being updated.", true
	}
}
