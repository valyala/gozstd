libzstd.a:
	cd zstd/lib && ZSTD_LEGACY_SUPPORT=0 make libzstd.a

clean:
	cd zstd && make clean
