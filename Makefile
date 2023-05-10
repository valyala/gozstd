GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOOS_GOARCH := $(GOOS)_$(GOARCH)
GOOS_GOARCH_NATIVE := $(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)
LIBZSTD_NAME := libzstd_$(GOOS_GOARCH).a
ZSTD_VERSION ?= v1.5.5
ZIG_BUILDER_IMAGE=euantorano/zig:0.10.1
BUILDER_IMAGE := local/builder_musl:2.0.0-$(shell echo $(ZIG_BUILDER_IMAGE) | tr : _ | tr / _)-1

.PHONY: libzstd.a $(LIBZSTD_NAME)

libzstd.a: $(LIBZSTD_NAME)
$(LIBZSTD_NAME):
ifeq ($(GOOS_GOARCH),$(GOOS_GOARCH_NATIVE))
	cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 MOREFLAGS=$(MOREFLAGS) $(MAKE) clean libzstd.a
	mv zstd/lib/libzstd.a $(LIBZSTD_NAME)
else
ifeq ($(GOOS_GOARCH),linux_arm)
	cd zstd/lib && CC=arm-linux-gnueabi-gcc ZSTD_LEGACY_SUPPORT=0 MOREFLAGS=$(MOREFLAGS) $(MAKE) clean libzstd.a
	mv zstd/lib/libzstd.a libzstd_linux_arm.a
endif
ifeq ($(GOOS_GOARCH),linux_arm64)
	cd zstd/lib && CC=aarch64-linux-gnu-gcc ZSTD_LEGACY_SUPPORT=0 MOREFLAGS=$(MOREFLAGS) $(MAKE) clean libzstd.a
	mv zstd/lib/libzstd.a libzstd_linux_arm64.a
endif
ifeq ($(GOOS_GOARCH),linux_musl_amd64)
	cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 MOREFLAGS=$(MOREFLAGS) $(MAKE) clean libzstd.a
	mv zstd/lib/libzstd.a libzstd_linux_musl_amd64.a
endif
ifeq ($(GOOS_GOARCH),linux_musl_arm64)
	cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 MOREFLAGS=$(MOREFLAGS) $(MAKE) clean libzstd.a
	mv zstd/lib/libzstd.a libzstd_linux_musl_arm64.a
endif
endif

package-builder:
	(docker image ls --format '{{.Repository}}:{{.Tag}}' | grep -q '$(BUILDER_IMAGE)$$') \
		|| docker build \
			--build-arg builder_image=$(ZIG_BUILDER_IMAGE) \
			--tag $(BUILDER_IMAGE) \
			builder

package-arch: package-builder
	docker run --rm \
		--mount type=bind,src="$(shell pwd)",dst=/zstd \
		-w /zstd \
		$(DOCKER_OPTS) \
		$(BUILDER_IMAGE) \
		-c 'cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 AR="zig ar" CC="zig cc -target $(TARGET_ARCH)-$(TARGET)" CXX="zig cc -target x86_64-$(TARGET)" MOREFLAGS=$(MOREFLAGS) make clean libzstd.a'
	mv zstd/lib/libzstd.a libzstd_$(OS)_$(ARCH).a

# freebsd and illumos aren't supported by zig compiler atm.
release:
	TARGET=macos TARGET_ARCH=aarch64 OS=darwin ARCH=arm64 $(MAKE) package-arch
	TARGET=macos TARGET_ARCH=x86_64 OS=darwin ARCH=amd64 $(MAKE) package-arch
	TARGET=windows TARGET_ARCH=x86_64 OS=windows ARCH=amd64 $(MAKE) package-arch
	TARGET=linux TARGET_ARCH=x86_64 OS=linux ARCH=amd64 $(MAKE) package-arch
	TARGET=linux TARGET_ARCH=aarch64 OS=linux ARCH=arm64 $(MAKE) package-arch
	TARGET=linux_musl TARGET_ARCH=x86_64 OS=linux-musl ARCH=amd64 $(MAKE) package-arch
	TARGET=linux_musl TARGET_ARCH=aarch64 OS=linux-musl ARCH=arm64 $(MAKE) package-arch

clean:
	rm -f $(LIBZSTD_NAME)
	cd zstd && $(MAKE) clean

update-zstd:
	rm -rf zstd-tmp
	git clone --branch $(ZSTD_VERSION) --depth 1 https://github.com/Facebook/zstd zstd-tmp
	rm -rf zstd-tmp/.git
	rm -rf zstd
	mv zstd-tmp zstd
	$(MAKE) clean libzstd.a
	cp zstd/lib/zstd.h .
	cp zstd/lib/zdict.h .
	cp zstd/lib/zstd_errors.h .

test:
	CGO_ENABLED=1 GODEBUG=cgocheck=2 go test -v

bench:
	CGO_ENABLED=1 go test -bench=.
