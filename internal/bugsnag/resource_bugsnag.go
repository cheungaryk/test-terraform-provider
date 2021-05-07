package bugsnag

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		UpdateContext: resourceProjectUpdate,
		DeleteContext: resourceProjectDelete,
		Schema:        getProjectSchema(true, true, true),
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Get("name").(string)
	project_type := d.Get("type").(string)
	ignore_old_browsers := d.Get("ignore_old_browsers").(bool)

	projects, diags := c.listProjects()
	if len(diags) > 0 {
		return diags
	}

	for _, project := range projects {
		if project["name"] == name {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "project already exists",
				Detail:   fmt.Sprintf(`the project %s already exists!`, name),
			})

			return diags
		}
	}

	projectID, diags := c.createProject(name, project_type, ignore_old_browsers)
	if len(diags) > 0 {
		return diags
	}

	d.SetId(projectID)
	resourceProjectRead(ctx, d, m)
	return diags
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	projectID := d.Id()

	project, diags := c.getProject(projectID)
	if len(diags) > 0 {
		return diags
	}

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "test",
		Detail:   fmt.Sprintf("hello %s", project),
	})

	for v := range getProjectSchema(true, false, true) {
		if err := d.Set(v, project[v]); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "error reading project state",
				Detail: fmt.Sprintf(`error message: %v
project: %v`, err, project),
			})
			return diags
		}
	}

	return diags
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	return diags
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	return diags
}
