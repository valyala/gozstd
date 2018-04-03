libzstd.a:
	cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 make libzstd.a
	mv zstd/lib/libzstd.a .
	cd zstd && make clean

clean:
	rm -f libzstd.a

update-zstd:
	rm -rf zstd-tmp
	git clone https://github.com/Facebook/zstd zstd-tmp
	rm -rf zstd-tmp/.git
	rm -rf zstd
	mv zstd-tmp zstd
	make libzstd.a
