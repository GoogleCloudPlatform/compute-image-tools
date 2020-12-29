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

// network.tf creates a network stack using NAT. For more info, see:
//    https://cloud.google.com/nat/docs/gce-example

resource "google_compute_network" "nat" {
  name                    = "nat"
  project                 = google_project.project.project_id
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "nat" {
  name          = "nat"
  project       = google_project.project.project_id
  network       = google_compute_network.nat.self_link
  ip_cidr_range = "10.0.0.0/16"
}

resource "google_compute_router" "nat" {
  name    = "nat"
  network = google_compute_network.nat.self_link
  project = google_project.project.project_id
}

resource "google_compute_router_nat" "nat" {
  name                               = "nat"
  router                             = google_compute_router.nat.name
  project                            = google_project.project.project_id
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}
