//go:build rocksdb

package corruption_test

import (
	"bytes"
	"fmt"
	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestRocksDBWALCorruption(t *testing.T) {
	const numKeys = 1000

	dir, err := os.MkdirTemp("", "rocksdb-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	log := grocksdb.NewStderrLogger(grocksdb.WarnInfoLogLevel, "ROCKSDB")
	defer log.Destroy()

	dbOptions := grocksdb.NewDefaultOptions()
	dbOptions.SetCreateIfMissing(true)
	dbOptions.SetParanoidChecks(true)
	dbOptions.DisabledAutoCompactions()
	dbOptions.SetInfoLog(log)

	// Open the database and write some data
	db, err := grocksdb.OpenDb(dbOptions, dir)
	require.NoError(t, err)
	opts := grocksdb.NewDefaultWriteOptions()
	opts.SetSync(true)

	for j := 0; j < numKeys; j++ {
		key := []byte(fmt.Sprintf("key-%d", j))
		value := []byte(fmt.Sprintf("value-%d", j))
		err := db.Put(opts, key, value)
		require.NoError(t, err)
	}

	verifyRocksDBIntegrity(t, db, numKeys)

	// Find the WAL file
	walFile, err := findWALFile(dir)
	require.NoError(t, err)

	// Corrupt the WAL file by flipping a random byte
	corruptWALFile(t, walFile)

	// Force Flush WAL to sstable
	db.Close()

	db, err = grocksdb.OpenDb(dbOptions, dir)
	require.NoError(t, err)
	verifyRocksDBIntegrity(t, db, numKeys)
	db.Close()
}

func TestRocksDBSSTCorruption(t *testing.T) {
	const numKeys = 1000

	dir, err := os.MkdirTemp("", "rocksdb-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	log := grocksdb.NewStderrLogger(grocksdb.WarnInfoLogLevel, "ROCKSDB")
	defer log.Destroy()

	dbOptions := grocksdb.NewDefaultOptions()
	dbOptions.SetCreateIfMissing(true)
	//dbOptions.SetParanoidChecks(true)
	dbOptions.DisabledAutoCompactions()
	dbOptions.SetInfoLog(log)

	// Open the database and write some data
	db, err := grocksdb.OpenDb(dbOptions, dir)
	require.NoError(t, err)
	opts := grocksdb.NewDefaultWriteOptions()
	opts.SetSync(true)

	for j := 0; j < numKeys; j++ {
		key := []byte(fmt.Sprintf("key-%d", j))
		value := []byte(fmt.Sprintf("value-%d", j))
		err := db.Put(opts, key, value)
		require.NoError(t, err)
	}
	// Force Flush WAL to sstable
	db.Close()
	db, err = grocksdb.OpenDb(dbOptions, dir)
	require.NoError(t, err)
	verifyRocksDBIntegrity(t, db, numKeys)
	db.Close()

	// Find the SST file
	sstFile, err := findSSTFile(dir)
	require.NoError(t, err)
	fmt.Printf("sstFile: %s\n", sstFile)

	for i := 0; i < numKeys; i++ {
		// Corrupt the SST file by flipping a random byte
		// Offset 9703 results in a checksum mismatch if  SetParanoidChecks(true)
		//corruptSSTFile(t, sstFile, 9703)
		// 8685
		corruptSSTFile(t, sstFile, 0)

		db, err = grocksdb.OpenDb(dbOptions, dir)
		require.NoError(t, err)
		verifyRocksDBIntegrity(t, db, numKeys)
		db.Close()
		time.Sleep(500 * time.Millisecond)
	}
}

func verifyRocksDBIntegrity(t *testing.T, db *grocksdb.DB, numKeys int) {
	opts := grocksdb.NewDefaultReadOptions()

	// Verify data integrity
	for j := 0; j < numKeys; j++ {
		key := []byte(fmt.Sprintf("key-%d", j))
		value, err := db.Get(opts, key)
		if err != nil {
			t.Logf("Failed to read key %s: %v", key, err)
			continue
		}
		expectedValue := []byte(fmt.Sprintf("value-%d", j))
		if !bytes.Equal(value.Data(), expectedValue) {
			t.Errorf("Unexpected value for key '%s': got '%s' want '%s'", key, value.Data(), expectedValue)
		}
		value.Free()
	}
}
