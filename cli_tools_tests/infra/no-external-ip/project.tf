//  Copyright 2020 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// project.tf creates a hardened project:
//  1. No default network
//  2. Disallow external IPs for VMs
//  3. Whitelist of enabled APIs

// Provides a random suffix when creating the project ID.
resource "random_id" "id" {
  byte_length = 3
  prefix      = "${var.project_prefix}-"
}

// Create a new GCP project.
//
// The project ID is truncated to 30 characters, as specified by
//  https://cloud.google.com/resource-manager/reference/rest/v1/projects
resource "google_project" "project" {
  name                = var.project_prefix
  project_id          = substr(random_id.id.hex, 0, 30)
  billing_account     = var.billing_account
  org_id              = var.organization_id
  auto_create_network = false
}

// Disable external APIs on GCE instances.
resource "google_project_organization_policy" "no_external_ip" {
  project    = google_project.project.project_id
  constraint = "compute.vmExternalIpAccess"

  list_policy {
    deny {
      all = true
    }
  }
}

// Enable APIs.
resource "google_project_service" "enabled_apis" {
  for_each = toset([
    "compute.googleapis.com",
    "cloudbuild.googleapis.com",
    "logging.googleapis.com",
  ])
  project            = google_project.project.project_id
  service            = each.key
  disable_on_destroy = false
}
