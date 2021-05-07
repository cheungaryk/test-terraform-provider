package bugsnag

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"organization_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("BUGSNAG_ORGANIZATION_ID", nil),
				},
				"api_token": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("BUGSNAG_API_TOKEN", nil),
				},
			},
			ResourcesMap: map[string]*schema.Resource{
				"bugsnag_project": resourceProject(),
			},
			DataSourcesMap: map[string]*schema.Resource{
				"bugsnag_projects": dataSourceProjects(),
				"bugsnag_project":  dataSourceProject(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type apiClient struct {
	HostURL        string
	HTTPapiClient  *http.Client
	OrganizationID string
	APIToken       string
}

// NewapiClient -
func NewapiClient(apiToken, organizationID string) *apiClient {
	return &apiClient{
		HTTPapiClient: &http.Client{Timeout: 10 * time.Second},
		HostURL:       fmt.Sprintf("%s/%s", HostURL, organizationID),
		APIToken:      apiToken,
	}
}

func (c *apiClient) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.APIToken))
	return c.HTTPapiClient.Do(req)
}

func (c *apiClient) testAuth() (*http.Response, error) {
	req, err := http.NewRequest("GET", c.HostURL, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(req)
}

func (c *apiClient) listProjects() ([]map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/projects?per_page=100", c.HostURL), nil)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	r, err := c.doRequest(req)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	// https://bugsnagapiv2.docs.apiary.io/#reference/projects/projects/list-an-organization's-projects
	if r.StatusCode != 200 {
		switch r.StatusCode {
		case 429:
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "rate limit reached",
				Detail: `You have reached the rate limit, please try again later.
For further, see https://bugsnagapiv2.docs.apiary.io/#introduction/rate-limiting.`,
			})
			return nil, diags
		default:
			defer r.Body.Close()

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err.Error())
			}

			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "unexpected error",
				Detail: fmt.Sprintf(`You have encountered an unexpected error.
Please see https://bugsnagapiv2.docs.apiary.io/#reference/projects/projects/list-an-organization's-projects for further information
error message: %s`, string(body)),
			})
			return nil, diags
		}
	}

	defer r.Body.Close()

	projects := make([]map[string]interface{}, 0)
	err = json.NewDecoder(r.Body).Decode(&projects)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return projects, diags
}

func (c *apiClient) getProject(projectID string) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/projects/%s", c.HostURL, projectID), nil)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	r, err := c.doRequest(req)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	if r.StatusCode != 200 {
		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err.Error())
		}

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "unexpected error",
			Detail: fmt.Sprintf(`You have encountered an unexpected error.
Please see https://bugsnagapiv2.docs.apiary.io/#reference/projects/projects/create-a-project-in-an-organization for further information
error message: %s`, string(body)),
		})
		return nil, diags
	}

	defer r.Body.Close()

	project := make(map[string]interface{}, 0)
	err = json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return project, diags
}

func (c *apiClient) createProject(name, projectType string, ignore_old_browsers bool) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	url_params := fmt.Sprintf("?name=%s&type=%s&ignore_old_browsers=%v", name, projectType, ignore_old_browsers)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/projects%s", c.HostURL, url_params), nil)
	if err != nil {
		return "", diag.FromErr(err)
	}

	r, err := c.doRequest(req)
	if err != nil {
		return "", diag.FromErr(err)
	}

	if r.StatusCode != 200 {
		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err.Error())
		}

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "unexpected error",
			Detail: fmt.Sprintf(`You have encountered an unexpected error.
Please see https://bugsnagapiv2.docs.apiary.io/#reference/projects/projects/create-a-project-in-an-organization for further information
error message: %s`, string(body)),
		})
		return "", diags
	}

	defer r.Body.Close()

	project := make(map[string]interface{}, 0)
	err = json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		return "", diag.FromErr(err)
	}

	id := project["id"].(string)

	if len(id) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "no project ID retrieved",
			Detail: fmt.Sprintf(`no project ID was retrieved.
received response body: %v`, project),
		})
		return "", diags
	}

	return id, diags
}

func (c *apiClient) updateProject(name, projectType string, ignore_old_browsers bool) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	url_params := fmt.Sprintf("?name=%s&type=%s&ignore_old_browsers=%v", name, projectType, ignore_old_browsers)

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/projects%s", c.HostURL, url_params), nil)
	if err != nil {
		return "", diag.FromErr(err)
	}

	r, err := c.doRequest(req)
	if err != nil {
		return "", diag.FromErr(err)
	}

	if r.StatusCode != 200 {
		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err.Error())
		}

		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "unexpected error",
			Detail: fmt.Sprintf(`You have encountered an unexpected error.
Please see https://bugsnagapiv2.docs.apiary.io/#reference/projects/projects/create-a-project-in-an-organization for further information
error message: %s`, string(body)),
		})
		return "", diags
	}

	defer r.Body.Close()

	project := make(map[string]interface{}, 0)
	err = json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		return "", diag.FromErr(err)
	}

	id := project["id"].(string)

	if len(id) == 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "no project ID retrieved",
			Detail: fmt.Sprintf(`no project ID was retrieved.
received response body: %v`, project),
		})
		return "", diags
	}

	return id, diags
}

func configure(version string, p *schema.Provider) func(c context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(c context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var diags diag.Diagnostics

		apiToken := d.Get("api_token").(string)
		if apiToken == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Bugsnag API Token not provided",
				Detail: `You did not provide the Bugsnag API token used for authentication. 
Please export the API token's value to $BUGSNAG_API_TOKEN.
For further, see https://bugsnagapiv2.docs.apiary.io/#introduction/authentication`,
			})
			return nil, diags
		}

		organizationID := d.Get("organization_id").(string)
		if organizationID == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Bugsnag organization ID not provided",
				Detail: `You did not provide the Bugsnag organization ID.
To get the value, ask your administrator or send an authenticated request to https://api.bugsnag.com/user/organizations.
Please provide it in the provider block or export the API token's value to $BUGSNAG_ORGANIZATION_ID.
For further, see https://bugsnagapiv2.docs.apiary.io/#reference/current-user/organizations/list-the-current-user's-organizations.`,
			})
			return nil, diags
		}

		client := NewapiClient(apiToken, organizationID)
		r, err := client.testAuth()
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to authenticate to Bugsnag",
				Detail:   fmt.Sprintf(`Unexpected error: %s`, err),
			})
			return nil, diags
		} else if r.StatusCode == 429 {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "API rate limit exceeded",
				Detail: `You have reached Bugsnag's API rate limit.
Please wait a moment and try again.`,
			})
			return nil, diags
		} else if r.StatusCode != 200 {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to authenticate to Bugsnag",
				Detail: fmt.Sprintf(`Unable to authenticate to Bugsnag API (%s) with the provided API token.
Please check that your token is valid and try again.`, client.HostURL),
			})
			return nil, diags
		}

		return c, diags
	}
}
