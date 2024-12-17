package corruption_test

import (
	"bytes"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestPebbleWALCorruption(t *testing.T) {
	const numKeys = 1000

	dir, err := os.MkdirTemp("", "pebble-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// Open the database and write some data
	db, err := pebble.Open(dir, &pebble.Options{})
	require.NoError(t, err)

	for j := 0; j < numKeys; j++ {
		key := []byte(fmt.Sprintf("key-%d", j))
		value := []byte(fmt.Sprintf("value-%d", j))
		err := db.Set(key, value, nil)
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())

	//verifyPebbleIntegrity(t, dir, numKeys)

	// Find the WAL file
	walFile, err := findWALFile(dir)
	require.NoError(t, err)

	// Corrupt the WAL file by flipping a random byte
	corruptWALFile(t, walFile)

	verifyPebbleIntegrity(t, dir, numKeys)
}

func TestPebbleSSTCorruption(t *testing.T) {
	const numKeys = 1000
	dir, err := os.MkdirTemp("", "pebble-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// Open the database and write some data
	db, err := pebble.Open(dir, &pebble.Options{})
	require.NoError(t, err)

	for j := 0; j < numKeys; j++ {
		key := []byte(fmt.Sprintf("key-%d", j))
		value := []byte(fmt.Sprintf("value-%d", j))
		err := db.Set(key, value, nil)
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())

	// Reopening the database appears to force a WAL flush
	// (likely a better way to do this, but I don't know pebble)
	verifyPebbleIntegrity(t, dir, numKeys)

	dbFile, err := findSSTFile(dir)
	require.NoError(t, err)

	// Corrupt the database file by flipping a random byte
	// Offset 3200 always results in a checksum miss match for the SSTable
	corruptSSTFile(t, dbFile, 3200)

	verifyPebbleIntegrity(t, dir, numKeys)

	// Open the database and write MORE data to see if corruption lingers or if new data
	// can be written into a corrupt database.
	fmt.Printf("Writing to corrupt database file\n")
	db, err = pebble.Open(dir, &pebble.Options{})
	require.NoError(t, err)

	for j := numKeys; j < numKeys+1000; j++ {
		key := []byte(fmt.Sprintf("key-%d", j))
		value := []byte(fmt.Sprintf("value-%d", j))
		err := db.Set(key, value, nil)
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())

	fmt.Printf("Verifying corrupt database file\n")
	verifyPebbleIntegrity(t, dir, numKeys+1000)

	// Force compaction over corrupted keys
	db, err = pebble.Open(dir, &pebble.Options{})
	require.NoError(t, err)
	assert.NoError(t, db.Compact([]byte("key-0"), []byte("key-1000"), false))
	require.NoError(t, db.Close())

	fmt.Printf("Verifying corrupt database file again\n")
	verifyPebbleIntegrity(t, dir, numKeys+1000)
}

func verifyPebbleIntegrity(t *testing.T, dir string, numKeys int) {
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Logf("Failed to open corrupted database: %v", err)
		return
	}
	defer db.Close()

	// Verify data integrity
	for j := 0; j < numKeys; j++ {
		key := []byte(fmt.Sprintf("key-%d", j))
		value, closer, err := db.Get(key)
		if err != nil {
			t.Logf("Failed to read key %s: %v", key, err)
			continue
		}
		expectedValue := []byte(fmt.Sprintf("value-%d", j))
		if !bytes.Equal(value, expectedValue) {
			t.Errorf("Unexpected value for key %s: got %s, want %s", key, value, expectedValue)
		}
		closer.Close()
	}
}

func findWALFile(dir string) (string, error) {
	var walFile string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//fmt.Printf("File: %s Size: %d\n", path, info.Size())
		if filepath.Ext(path) == ".log" {
			walFile = path
			return filepath.SkipAll
		}
		return nil
	})
	if walFile == "" {
		return "", fmt.Errorf("WAL file not found")
	}
	return walFile, err
}

func corruptWALFile(t *testing.T, filename string) {
	t.Helper()

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.False(t, len(data) == 0)

	// Choose a random byte to corrupt
	byteIndex := rand.Intn(len(data))
	byteIndex = 36311 // This index always results in truncation of the WAL

	t.Logf("Corrupting WAL Offset: %d", byteIndex)
	data[byteIndex] = 0xFF
	require.NoError(t, os.WriteFile(filename, data, 0644))
}

func findSSTFile(dir string) (string, error) {
	var dbFile string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Printf("File: %s Size: %d\n", path, info.Size())
		if filepath.Ext(path) == ".sst" {
			fmt.Printf("file: %s Size: %d\n", path, info.Size())
			dbFile = path
			return filepath.SkipAll
		}
		return nil
	})
	if dbFile == "" {
		return "", fmt.Errorf("sst file not found")
	}
	return dbFile, err
}

func corruptSSTFile(t *testing.T, filename string, offset int) {
	t.Helper()

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.False(t, len(data) == 0)

	byteIndex := offset
	if offset == 0 {
		// Choose a random byte and bit to flip
		byteIndex = rand.Intn(len(data))
	}

	t.Logf("Corrupting Database Offset: %d", byteIndex)
	data[byteIndex] = 0xFF
	require.NoError(t, os.WriteFile(filename, data, 0644))
}
