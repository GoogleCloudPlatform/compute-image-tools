//  Copyright 2019 Google Inc. All Rights Reserved.
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
//  limitations under the License

package compute

import (
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

// ParseNodeAffinityLabels parses sole tenant affinities
// labels - array of strings with node affinity label info. Each label is of the following format:
// <key>,<operator>,<value>,<value2>... where <operator> can be one of: IN, NOT.
// For example: workload,IN,prod,test is a label with key 'workload' and values 'prod' and 'test'
func ParseNodeAffinityLabels(labels []string) ([]*compute.SchedulingNodeAffinity, []*computeBeta.SchedulingNodeAffinity, error) {
	var nodeAffinities []*compute.SchedulingNodeAffinity
	var nodeAffinitiesBeta []*computeBeta.SchedulingNodeAffinity

	for _, label := range labels {
		labelParts := strings.Split(label, ",")
		if len(labelParts) < 3 {
			return nil, nil, daisy.Errf(
				"node affinity label `%v` should be of <key>,<operator>,<values> format", label)
		}
		key := strings.TrimSpace(labelParts[0])
		if key == "" {
			return nil, nil, daisy.Errf("affinity label key cannot be empty")
		}
		operator := strings.TrimSpace(labelParts[1])
		if operator != "IN" && operator != "NOT_IN" && operator != "OPERATOR_UNSPECIFIED" {
			return nil, nil, daisy.Errf(
				"node affinity label operator should be one of: `IN`, `NOT_IN` or `OPERATOR_UNSPECIFIED`, but instead received `%v`",
				operator,
			)
		}

		values := labelParts[2:]
		for i, value := range values {
			values[i] = strings.TrimSpace(value)
			if values[i] == "" {
				return nil, nil, daisy.Errf("affinity label value cannot be empty")
			}
		}
		nodeAffinities = append(nodeAffinities, &compute.SchedulingNodeAffinity{
			Key:      key,
			Operator: operator,
			Values:   values,
		})
		nodeAffinitiesBeta = append(nodeAffinitiesBeta, &computeBeta.SchedulingNodeAffinity{
			Key:      key,
			Operator: operator,
			Values:   values,
		})

	}
	return nodeAffinities, nodeAffinitiesBeta, nil
}
