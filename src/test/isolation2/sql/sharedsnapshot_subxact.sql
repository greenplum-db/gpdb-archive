-- This test case is used to test if sharedsnapshot can synced
-- subxact infomations between writer gang and reader gang.

1: CREATE TABLE test_sharedsnapshot_subxact_1(c1 int, c2 int);
1: CREATE TABLE test_sharedsnapshot_subxact_2(c1 int, c2 int);

1: INSERT INTO test_sharedsnapshot_subxact_1 VALUES (1,2),(3,4);

2: BEGIN;
2: INSERT INTO test_sharedsnapshot_subxact_2 VALUES (1,2);
-- start a sub transaction
2: SAVEPOINT p1;
2: INSERT INTO test_sharedsnapshot_subxact_2 VALUES (3,4);

-- Advance ShmemVariableCache->latestCompletedXid, so that
-- session 2's transacion id can be put in xip.
3: CREATE TABLE test_sharedsnapshot_subxact_3(c1 int, c2 int);
3: BEGIN;
3: DROP TABLE test_sharedsnapshot_subxact_3;

2: SELECT * FROM test_sharedsnapshot_subxact_2;

-- Issue a query contains reader gang to see if this reader
-- gang will used correct snapshot to scan test_sharedsnapshot_subxact_2.
1: SELECT * FROM test_sharedsnapshot_subxact_1 as t1 left join test_sharedsnapshot_subxact_2 as t2 on t1.c1=t2.c2;

2: COMMIT;
-- Check if tuple (3,4) is visible and won't be set xmin to invalid by
-- session 1's reader gang.
2: SELECT * FROM test_sharedsnapshot_subxact_2;


3: COMMIT;

-- Clean up
1: DROP TABLE test_sharedsnapshot_subxact_1;
1: DROP TABLE test_sharedsnapshot_subxact_2;

-- Test from Github Issue 17275
create table t1_17275(a int);
create table t2_17275(a int);

1: begin;
-- use is not null to make test stable
1: select txid_current() is not null;
1: end;

2: begin;
2: select txid_current() is not null;

3: begin;
3: savepoint s1;
3: truncate t2_17275;

4: begin;
4: select txid_current() is not null;
4: end;

5: begin;
5: select txid_current() is not null;

6: select * from t1_17275 join (select oid from pg_class) x(a) on x.a = t1_17275.a;

3: savepoint s2;
3: truncate t2_17275;

1q:
2q:
3q:
4q:
5q:
6q:

drop table t1_17275;
drop table t2_17275;
