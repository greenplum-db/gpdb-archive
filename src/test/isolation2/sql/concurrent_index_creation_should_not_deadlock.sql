-- Test to make sure non-first concurrent index creations don't deadlock
-- Create an append only table, popluated with data
CREATE TABLE index_deadlocking_test_table (value int) WITH (appendonly=true);
CREATE INDEX index_deadlocking_test_table_initial_index on index_deadlocking_test_table (value);

-- Setup a fault to ensure that both sessions pauses while creating an index,
-- ensuring a concurrent index creation.
SELECT gp_inject_fault('defineindex_before_acquire_lock', 'suspend', 1);

-- Attempt to concurrently create an index
1>: CREATE INDEX index_deadlocking_test_table_idx1 ON index_deadlocking_test_table (value);
2>: CREATE INDEX index_deadlocking_test_table_idx2 ON index_deadlocking_test_table (value);
SELECT gp_wait_until_triggered_fault('defineindex_before_acquire_lock', 2, 1);
SELECT gp_inject_fault('defineindex_before_acquire_lock', 'reset', 1);

-- Both index creation attempts should succeed
1<:
2<:
