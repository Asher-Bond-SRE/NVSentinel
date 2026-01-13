# Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Main Makefile for NVIDIA Device API
# Delegates to api/Makefile for all operations

.PHONY: all
all: ## Build and generate all (delegates to api/)
	$(MAKE) -C api all

.PHONY: build
build: ## Build the Go module (delegates to api/)
	$(MAKE) -C api build

.PHONY: protos-generate
protos-generate: ## Generate Go code from Proto definitions (delegates to api/)
	$(MAKE) -C api protos-generate

.PHONY: protos-clean
protos-clean: ## Remove generated code (delegates to api/)
	$(MAKE) -C api protos-clean

.PHONY: lint
lint: ## Run linting checks
	cd api && go vet ./...

.PHONY: test
test: ## Run tests
	cd api && go test ./...

.PHONY: clean
clean: protos-clean ## Clean all generated and build artifacts
	rm -rf api/bin/

.PHONY: help
help: ## Display this help
	@echo "NVIDIA Device API Makefile"
	@echo ""
	@echo "Available targets:"
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort -u | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
