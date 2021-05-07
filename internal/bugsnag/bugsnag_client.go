package bugsnag

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

const HostURL string = "https://api.bugsnag.com/organizations"

// Client -
type Client struct {
	HostURL        string
	HTTPClient     *http.Client
	OrganizationID string
	APIToken       string
}

// NewClient -
func NewClient(apiToken, organizationID string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		HostURL:    fmt.Sprintf("%s/%s", HostURL, organizationID),
		APIToken:   apiToken,
	}
}

func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.APIToken))
	return c.HTTPClient.Do(req)
}

func (c *Client) testAuth() (*http.Response, error) {
	req, err := http.NewRequest("GET", c.HostURL, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(req)
}

func (c *Client) listProjects() ([]map[string]interface{}, diag.Diagnostics) {
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

func (c *Client) getProject(projectID string) (map[string]interface{}, diag.Diagnostics) {
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

func (c *Client) createProject(name, projectType string, ignore_old_browsers bool) (string, diag.Diagnostics) {
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

func (c *Client) updateProject(name, projectType string, ignore_old_browsers bool) (string, diag.Diagnostics) {
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
