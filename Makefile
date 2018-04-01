libzstd.a:
	cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 make libzstd.a

clean:
	cd zstd && make clean

update-zstd:
	rm -rf zstd-tmp
	git clone https://github.com/Facebook/zstd zstd-tmp
	rm -rf zstd-tmp/.git
	mv zstd/.gitignore zstd-tmp/
	rm -rf zstd
	mv zstd-tmp zstd
	make libzstd.a
