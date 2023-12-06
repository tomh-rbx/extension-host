# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## licenses-report: generate a report of all licenses
.PHONY: licenses-report
licenses-report:
ifeq ($(SKIP_LICENSES_REPORT), true)
	@echo "Skipping licenses report"
	rm -rf ./licenses && mkdir -p ./licenses
else
	@echo "Generating licenses report"
	rm -rf ./licenses
	go run github.com/google/go-licenses@v1.6.0 save . --save_path ./licenses
	go run github.com/google/go-licenses@v1.6.0 report . > ./licenses/THIRD-PARTY.csv
	cp LICENSE ./licenses/LICENSE.txt
endif

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: prepare_audit
prepare_audit:
	echo Enable KVM group perms
	echo 'KERNEL=="kvm", GROUP="kvm", MODE="0666", OPTIONS+="static_node=kvm"' | sudo tee /etc/udev/rules.d/99-kvm4all.rules
	sudo udevadm control --reload-rules
	sudo udevadm trigger --name-match=kvm
	sudo apt-get update
	sudo apt-get install -y libvirt-clients libvirt-daemon-system libvirt-daemon virtinst bridge-utils qemu qemu-system-x86
	sudo usermod -a -G kvm,libvirt $USER
	minikube config set WantUpdateNotification false
	minikube config set cpus max
	minikube config set memory 8g

## audit: run quality control checks
.PHONY: audit
audit:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-ST1003,-U1000 ./...
	go test -race -vet=off -coverprofile=coverage.out -timeout 30m ./...

	go mod verify

## charttesting: Run Helm chart unit tests
.PHONY: charttesting
charttesting:
	@set -e; \
	for dir in charts/steadybit-extension-*; do \
		echo "Unit Testing $$dir"; \
		helm unittest $$dir; \
	done
## chartlint: Lint charts
.PHONY: chartlint
chartlint:
	ct lint --config chartTesting.yaml

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build: build the extension
.PHONY: build
build:
	goreleaser build --clean --snapshot --single-target -o extension

## run: run the extension
.PHONY: run
run: tidy build
	./extension

## container: build the container image
.PHONY: container
container:
	docker buildx build --build-arg BUILD_WITH_COVERAGE="true" --build-arg SKIP_LICENSES_REPORT="true" -t extension-host:latest --output=type=docker .

## linuxpkg: build the linux packages
.PHONY: linuxpkg
linuxpkg:
	goreleaser release --clean --snapshot --skip=sign
