.PHONY: build clean deploy

PATH := $(PATH):$(PWD)/node_modules/.bin

build:
	#env GOOS=linux go build -ldflags="-s -w" -o bin/hello hello/main.go
	env GOOS=linux go build -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/gamelist src/gamelist/*.go
	env GOOS=linux go build -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/gamedetails src/gamedetails/gamedetails.go

clean:
	rm -rf ./bin

deploy: node_modules clean build
	serverless deploy --verbose

.PHONY: shell
shell:
	nix-shell

.PHONY: deps
deps:
	go get github.com/aws/aws-lambda-go/lambda
	go get github.com/aws/aws-lambda-go/events
	go get github.com/aws/aws-sdk-go/aws
	go get -d ./src/

node_modules: package.json
	npm install
	touch node_modules

.PHONY: nix-%
nix-%:
	@echo "run inside nix-shell: $*"
	nix-shell --pure --run "$(MAKE) $*"

# Upgrade to the latest commit of the selected nixpkgs branch
.PHONY: upgrade
upgrade-nix: NIX_FILE=shell.nix
upgrade-nix:
	@echo "Updating nixpkgs from branch: $(NIX_BRANCH)"; \
	set -e pipefail; \
	rev=$$(curl https://api.github.com/repos/NixOS/nixpkgs-channels/branches/$(NIX_BRANCH) | jq -er .commit.sha); \
	echo "Updating nixpkgs to hash: $$rev"; \
	sha=$$(nix-prefetch-url --unpack https://github.com/NixOS/nixpkgs-channels/archive/$$rev.tar.gz); \
	sed -i \
		-e "2s|.*|    # $(NIX_BRANCH)|" \
		-e "3s|.*|    url = \"https://github.com/NixOS/nixpkgs-channels/archive/$$rev.tar.gz\";|" \
		-e "4s|.*|    sha256 = \"$$sha\";|" \
		$(NIX_FILE)

test:
	go test ./tests/
