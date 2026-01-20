// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconciler

import (
	"testing"

	"github.com/nvidia/nvsentinel/store-client/pkg/datastore"
	"github.com/stretchr/testify/assert"
)

func TestExtractReceivedAtTimestamp(t *testing.T) {
	tests := []struct {
		name         string
		event        datastore.Event
		expectZero   bool
		expectedUnix int64
		description  string
	}{
		{
			name: "valid int64 timestamp",
			event: datastore.Event{
				"_received_at": int64(1640000000),
			},
			expectZero:   false,
			expectedUnix: 1640000000,
			description:  "Should extract valid int64 timestamp",
		},
		{
			name: "missing timestamp",
			event: datastore.Event{
				"other_field": "value",
			},
			expectZero:  true,
			description: "Should return zero time when _received_at is missing",
		},
		{
			name: "wrong type - string",
			event: datastore.Event{
				"_received_at": "not a number",
			},
			expectZero:  true,
			description: "Should return zero time when _received_at has wrong type",
		},
		{
			name: "wrong type - float64",
			event: datastore.Event{
				"_received_at": float64(1640000000),
			},
			expectZero:  true,
			description: "Should return zero time when _received_at is float64 (not int64)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractReceivedAtTimestamp(tt.event)

			if tt.expectZero {
				assert.True(t, result.IsZero(), tt.description)
			} else {
				assert.False(t, result.IsZero(), tt.description)
				assert.Equal(t, tt.expectedUnix, result.Unix(), "Unix timestamp should match")
			}
		})
	}
}
