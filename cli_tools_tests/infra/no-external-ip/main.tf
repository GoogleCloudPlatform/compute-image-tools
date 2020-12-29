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

variable "wrapper_executors" {
  description = "Accounts that will execute binaries directly. The cloud build service account is included automatically."
  default = [
    "user:ericedens@google.com",
    "user:zoranl@google.com",
    "user:tzz@google.com",
  ]
}

variable "gcloud_executors" {
  description = "Accounts that will start workflows using gcloud."
  default = [
    "user:ericedens@google.com",
    "user:zoranl@google.com",
    "user:tzz@google.com",
  ]
}

variable "project_prefix" {
  description = "The prefix to use when creating the project. A random string is appended to avoid collisions."
  default     = "gce-guest-no-external-ip"
}

variable "billing_account" {
  description = "The billing account for the new project."
}

variable "organization_id" {
  description = "The organization for the new project."
}

variable "region" {
  description = "Region where network infrastructure is created."
  default     = "us-west1"
}

provider "google" {
  region = var.region
}

output "project_id" {
  value = google_project.project.project_id
}

output "network" {
  value = google_compute_network.nat.self_link
}

output "subnet" {
  value = google_compute_subnetwork.nat.self_link
}
