package platform

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/quiet-terminal-interactive/qtidocs/internal/mailer"
)

const deployTemplateURL = "https://github.com/quiet-terminal-interactive/qtidocs/blob/main/internal/registry/templates/deploy.yml"

var deploySecretEmailTmpl = template.Must(template.New("deploySecretEmail").Parse(`Hi,

{{.Subdomain}}.qtidocs.dev has been registered and its first build is on the way.

To rebuild on every push and get pull request previews, add the deploy workflow to {{.Repo}}.

Your deploy secret:

    {{.Secret}}

It is only sent again if the registry entry's repo or contact changes, which replaces it. Treat it like a password: anyone holding it can trigger builds of your site.

Setup
-----

1. Add the secret to {{.Repo}} as a repository secret named QTIDOCS_DEPLOY_SECRET.
   On GitHub: Settings -> Secrets and variables -> Actions -> New repository secret.
   Or with the GitHub CLI:

       gh secret set QTIDOCS_DEPLOY_SECRET --repo {{.OwnerName}}

   and paste the secret when prompted.

2. Copy the workflow template into your repo as .github/workflows/qtidocs.yml:

       {{.DeployTemplateURL}}

{{if .IsMainBranch}}   You publish from main, so use the file as it is.
{{else}}   You publish from {{printf "%q" .Branch}}, so change on.push.branches in the workflow from main to {{.Branch}}.
{{end}}
3. Push the workflow to {{.Branch}}. That push triggers a rebuild, and https://{{.Subdomain}}.qtidocs.dev updates shortly after.

If the secret leaks, or you didn't expect this email, open an issue at https://github.com/quiet-terminal-interactive/qtidocs/issues.

QTIDocs
`))

func deploySecretEmail(req registerRequest, secret string) (mailer.Message, error) {
	repo := strings.TrimPrefix(req.Repo, "https://")
	repo = strings.TrimPrefix(repo, "http://")
	repo = strings.TrimSuffix(repo, ".git")
	ownerName := strings.TrimPrefix(repo, "github.com/")

	var b strings.Builder
	data := struct {
		Subdomain         string
		Repo              string
		OwnerName         string
		Secret            string
		Branch            string
		IsMainBranch      bool
		DeployTemplateURL string
	}{
		Subdomain:         req.Subdomain,
		Repo:              repo,
		OwnerName:         ownerName,
		Secret:            secret,
		Branch:            req.Branch,
		IsMainBranch:      req.Branch == "main",
		DeployTemplateURL: deployTemplateURL,
	}
	if err := deploySecretEmailTmpl.Execute(&b, data); err != nil {
		return mailer.Message{}, fmt.Errorf("platform: rendering deploy-secret email: %w", err)
	}

	return mailer.Message{
		To:      req.Contact,
		Subject: fmt.Sprintf("Your QTIDocs deploy secret for %s.qtidocs.dev", req.Subdomain),
		Body:    b.String(),
	}, nil
}
