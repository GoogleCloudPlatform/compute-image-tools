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

// iam.tf configures a project's permissions to allow CLI tool executors
// with two different levels of permissions.

locals {
  // wrapper executors require the most permissions, since they will be creating,
  // reading, and destroying resources. In production, this actor is typically
  // the cloud build service account.
  wrapper_roles = [
    "roles/compute.admin",
    "roles/cloudbuild.builds.builder",
    "roles/iam.serviceAccountTokenCreator",
    "roles/iam.serviceAccountUser",
  ]
  gcloud_roles = [
    // gcloud executors have less permissions -- only those required for invoking
    //  `gcloud compute images import` or similar.
    "roles/storage.admin",
    "roles/viewer",
    "roles/resourcemanager.projectIamAdmin",
    "roles/cloudbuild.builds.editor",
  ]
}

// Apply wrapper roles.
resource "google_project_iam_binding" "wrapper" {
  for_each = toset(local.wrapper_roles)

  project = google_project.project.project_id
  role    = each.key

  members = concat(var.wrapper_executors, [
    "serviceAccount:${google_project.project.number}@cloudbuild.gserviceaccount.com"
  ])

  # Wait on the creation of the cloud build service account.
  depends_on = [
    google_project_service.enabled_apis,
  ]
}

// Apply gcloud roles.
resource "google_project_iam_binding" "gcloud" {
  for_each = toset(local.gcloud_roles)

  project = google_project.project.project_id
  role    = each.key

  members = var.gcloud_executors
}
