.PHONY: test
CGO_CFLAGS := -I/opt/homebrew/Cellar/rocksdb/9.8.4/include
CGO_LDFLAGS := -L/opt/homebrew/Cellar/rocksdb/9.8.4/lib -L/opt/homebrew/Cellar/zstd/1.5.6/lib -L/opt/homebrew/Cellar/lz4/1.9.4/lib -L/opt/homebrew/Cellar/snappy/1.2.1/lib -lrocksdb -lstdc++ -lm
GOOS=darwin
GOARCH=arm64
CGO_ENABLED=1

export CGO_CFLAGS
export CGO_LDFLAGS
export GOOS
export GOARCH
export CGO_ENABLED
export CC=clang
export CXX=clang++

# Install RockSDB With
# $ brew install rocksdb snappy lz4 zstd
test:
	@echo "CGO_CFLAGS: $(CGO_CFLAGS)"
	@echo "CGO_LDFLAGS: $(CGO_LDFLAGS)"
	#go test -v -p=1 -count=1 ./... -v -tags rocksdb -run ^TestRocksDB*$
	go test -v -p=1 -count=1 ./... -v -tags rocksdb -run ^TestRocksDBSSTCorruption$
	#go test -v -p=1 -count=1 ./... -v -tags rocksdb -run ^TestRocksDBWALCorruption$
