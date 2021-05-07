resource "bugsnag_project" "test" {
  name = "bugsnag-tf-test"
  type = "go"
}

output "project" {
  value = bugsnag_project.test
}