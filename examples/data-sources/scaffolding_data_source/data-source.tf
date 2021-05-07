data "bugsnag_projects" "all" {}

data "bugsnag_project" "test" {
  name = var.project_name
}