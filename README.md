# Pebble

### Pebble Corrupted WAL Test
Pebble truncates WAL entries when it finds corruption in the WAL with a warning, but the database remains available.
Any keys written to the database which occur after the corruption in the WAL are lost.

```
wal_corruption_test.go:44: Corrupted WAL Offset: 36311
2024/12/17 09:25:36 [JOB 1] WAL file /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/pebble-test1967837142/000002.log with log number 000002 stopped reading at offset: 36289; replayed 869 keys in 869 batches
wal_corruption_test.go:66: Failed to read key key-869: pebble: not found
wal_corruption_test.go:66: Failed to read key key-870: pebble: not found
wal_corruption_test.go:66: Failed to read key key-871: pebble: not found
wal_corruption_test.go:66: Failed to read key key-872: pebble: not found
wal_corruption_test.go:66: Failed to read key key-873: pebble: not found
wal_corruption_test.go:66: Failed to read key key-874: pebble: not found
-- SNIP --
```

### Pebble Corrupted sstable
Pebble returns the error `pebble/table: invalid table 000004 (checksum mismatch at 1615/1618)` for keys which are 
affected by the corruption. Other keys not affected by the corrupted table are returned successfully.

```
2024/12/17 09:49:45 [JOB 1] WAL file /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/pebble-test1733585767/000002.log with log number 000002 stopped reading at offset: 41791; replayed 1000 keys in 1000 batches
file: /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/pebble-test1733585767 Size: 288
file: /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/pebble-test1733585767/000004.sst Size: 9499
wal_corruption_test.go:84: Corrupting Database Offset: 3200
2024/12/17 09:49:45 [JOB 1] WAL file /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/pebble-test1733585767/000005.log with log number 000005 stopped reading at offset: 0; replayed 0 keys in 0 batches
wal_corruption_test.go:126: Failed to read key key-3: pebble/table: invalid table 000004 (checksum mismatch at 1615/1618)
wal_corruption_test.go:126: Failed to read key key-4: pebble/table: invalid table 000004 (checksum mismatch at 1615/1618)
wal_corruption_test.go:126: Failed to read key key-27: pebble/table: invalid table 000004 (checksum mismatch at 1615/1618)
wal_corruption_test.go:126: Failed to read key key-28: pebble/table: invalid table 000004 (checksum mismatch at 1615/1618)
wal_corruption_test.go:126: Failed to read key key-29: pebble/table: invalid table 000004 (checksum mismatch at 1615/1618)
wal_corruption_test.go:126: Failed to read key key-30: pebble/table: invalid table 000004 (checksum mismatch at 1615/1618)
-- SNIP --
```

### Pebble Conclusion
A corrupt database remains available, new values can be added to the database, but corrupted values cannot be updated.

From the CRDB mailing list https://groups.google.com/g/cockroach-db/c/aNUS04vsjPM from 2016
> Currently we merely detect corruption but do not take steps to fix it. We are considering schemes where we could use
> majority voting among replicas to restore a known-good version of the data in case one of the replicas is corrupted,
> however this is not implemented yet.


# RocksDB

### Corrupted WAL Test
RocksDB also truncates WAL entries when it finds corruption in the WAL. It logs a warning, but remains available.

```
=== RUN   TestRocksDBWALCorruption
rocksdb_test.go:51: Corrupting WAL Offset: 36311
2024/12/17-12:26:54.138735 17d3b7000 ROCKSDB[WARN] [db/db_impl/db_impl_open.cc:1119] /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/rocksdb-test3570066670/000004.log: dropping 1482 bytes; Corruption: checksum mismatch
rocksdb_test.go:130: Unexpected value for key 'key-961': got '' want 'value-961'
rocksdb_test.go:130: Unexpected value for key 'key-962': got '' want 'value-962'
rocksdb_test.go:130: Unexpected value for key 'key-963': got '' want 'value-963'
rocksdb_test.go:130: Unexpected value for key 'key-964': got '' want 'value-964'
rocksdb_test.go:130: Unexpected value for key 'key-965': got '' want 'value-965'
rocksdb_test.go:130: Unexpected value for key 'key-966': got '' want 'value-966'
rocksdb_test.go:130: Unexpected value for key 'key-967': got '' want 'value-967'
rocksdb_test.go:130: Unexpected value for key 'key-968': got '' want 'value-968'
rocksdb_test.go:130: Unexpected value for key 'key-969': got '' want 'value-969'
rocksdb_test.go:130: Unexpected value for key 'key-970': got '' want 'value-970'
rocksdb_test.go:130: Unexpected value for key 'key-971': got '' want 'value-971'
```

### Corrupted SST
RocksDB ignored more corruption than Pebble. Even after multiple random corruption attempts data verification passed.
I'm not sure how it achieved this feat, unless `db.Close()` doesn't completely unload the SSTable pages cached in
memory, which is likely an artifact of the `grocksdb` bindings and not rocksDB it's self.

In order to test this theory, I modified the test to create the data tables then `os.Exit()` the test. On subsequent
runs the test only read the key/values and verified the correct data. I had to open the SST and manually edit the file
with a HEX editor, changing the `value-26` to `value-X6` before I got a checksum error when fetching a key.

The database remained available in the face of corruption, as I was able to retrieve key values within the same 
table, but in different blocks where the corruption did not exist.
```
=== RUN   TestRocksDBSSTCorruption
rocksdb_test.go:132: Failed to read key key-3: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-4: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-27: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-28: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-29: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-30: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-31: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-32: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
rocksdb_test.go:132: Failed to read key key-33: Corruption: block checksum mismatch: stored(context removed) = 1930575595, computed = 3064321270, type = 4  in rocksdb/000008.sst offset 1613 size 1621
```

In most cases, opening a file with corrupt data was acceptable. However, I was able to corrupt the file enough
that `OpenDb()` returned a checksum error, reporting the manifest was corrupt.

```
=== RUN   TestRocksDBSSTCorruption
rocksdb_test.go:107: Corrupting Database Offset: 7218
rocksdb_test.go:107: Corrupting Database Offset: 8532
rocksdb_test.go:107: Corrupting Database Offset: 6555
rocksdb_test.go:107: Corrupting Database Offset: 1399
rocksdb_test.go:107: Corrupting Database Offset: 4732
rocksdb_test.go:107: Corrupting Database Offset: 4765
rocksdb_test.go:107: Corrupting Database Offset: 782
rocksdb_test.go:107: Corrupting Database Offset: 6976
rocksdb_test.go:107: Corrupting Database Offset: 305
rocksdb_test.go:107: Corrupting Database Offset: 2346
rocksdb_test.go:107: Corrupting Database Offset: 8707
2024/12/17-12:18:57.251788 17721b000 ROCKSDB[WARN] [db/db_impl/db_impl_open.cc:2311] DB::Open() failed: Corruption: block checksum mismatch: stored(context removed) = 3088797836, computed = 850204924, type = 4  in /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/rocksdb-test1967995396/000008.sst offset 8652 size 89  The file /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/rocksdb-test1967995396/MANIFEST-000050 may be corrupted.
rocksdb_test.go:110:
Error Trace:	/Users/thrawn/Development/corruption-testing/rocksdb_test.go:110
Error:      	Received unexpected error:
Corruption: block checksum mismatch: stored(context removed) = 3088797836, computed = 850204924, type = 4  in /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/rocksdb-test1967995396/000008.sst offset 8652 size 89  The file /var/folders/yj/mc5s1dgs4_59bc_q5j5wjp500000gn/T/rocksdb-test1967995396/MANIFEST-000050 may be corrupted.
Test:       	TestRocksDBSSTCorruption
```
