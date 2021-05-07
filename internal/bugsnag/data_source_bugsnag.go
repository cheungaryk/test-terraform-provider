package bugsnag

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func getProjectSchema(nameRequired bool, typeRequired bool, ignore_old_browsers bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Computed: !nameRequired,
			Required: nameRequired,
		},
		"global_grouping": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"location_grouping": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"discarded_app_versions": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"discarded_errors": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"url_whitelist": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"ignore_old_browsers": getIgnoreOldBrowsers(ignore_old_browsers),
		"ignored_browser_versions": {
			Type:     schema.TypeMap,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"resolve_on_deploy": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"organization_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"type": {
			Type:     schema.TypeString,
			Computed: !typeRequired,
			Required: typeRequired,
		},
		"slug": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"api_key": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"is_full_view": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"release_stages": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"language": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"created_at": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"updated_at": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"url": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"html_url": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"errors_url": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"events_url": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"open_error_count": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"for_review_error_count": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"collaborators_count": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"custom_event_fields_used": {
			Type:     schema.TypeInt,
			Computed: true,
		},
	}
}

func getIgnoreOldBrowsers(ignoreOldBrowsers bool) *schema.Schema {
	sch := schema.Schema{
		Type: schema.TypeBool,
	}

	if ignoreOldBrowsers {
		sch.Computed = true
	} else {
		sch.Required = true
	}

	return &sch
}

func dataSourceProjects() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectsRead,
		Schema: map[string]*schema.Schema{
			"projects": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: getProjectSchema(false, false, true),
				},
			},
		},
	}
}

func dataSourceProjectsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics
	projects, diags := client.listProjects()
	if len(diags) > 0 {
		return diags
	}

	if err := d.Set("projects", projects); err != nil {
		return diag.FromErr(err)
	}

	// always run
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

// single project
func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceProjectRead,
		Schema:      getProjectSchema(true, false, true),
	}
}

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	projects, diags := client.listProjects()
	if len(diags) > 0 {
		return diags
	}

	projectName := d.Get("name").(string)
	for _, project := range projects {
		if project["name"] == projectName {
			for v := range getProjectSchema(true, false, true) {
				if err := d.Set(v, project[v]); err != nil {
					return diag.FromErr(err)
				}
			}

			// always run
			d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

			return diags
		}
	}

	d.SetId("")
	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "unable to find projects with the provided name",
		Detail: fmt.Sprintf(`Unable to find the project with the name %s.
Please make sure that the project exists (or check your spelling) and try again.`, projectName),
	})
	return diags
}
