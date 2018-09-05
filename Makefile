libzstd_target = libzstd.a
ifeq ($(OS),Windows_NT)
	libzstd_target = libzstd_windows.a
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		libzstd_target = libzstd_linux.a
	endif
	ifeq ($(UNAME_S),Darwin)
		libzstd_target = libzstd_darwin.a
	endif
endif

libzstd.a:
	cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 make libzstd.a
	mv zstd/lib/libzstd.a $(libzstd_target)
	cd zstd && make clean

clean:
	rm -f $(libzstd_target)

update-zstd:
	rm -rf zstd-tmp
	git clone --branch master --depth 1 https://github.com/Facebook/zstd zstd-tmp
	rm -rf zstd-tmp/.git
	rm -rf zstd
	mv zstd-tmp zstd
	make libzstd.a
	cp zstd/lib/zstd.h .
	cp zstd/lib/dictBuilder/zdict.h .
	cp zstd/lib/common/zstd_errors.h .

test:
	GODEBUG=cgocheck=2 go test -v

bench:
	go test -bench=.
