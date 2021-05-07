terraform {
  required_providers {
    bugsnag = {
      version = "0.2"
      source  = "hashicorp.com/edu/bugsnag"
    }
  }
}

provider "bugsnag" {}