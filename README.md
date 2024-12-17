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

