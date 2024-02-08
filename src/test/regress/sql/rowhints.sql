-- Test Optimizer Row Hints Feature
--
-- Purpose: Test that row hints may be used to edit cardinality estimates

LOAD 'pg_hint_plan';

DROP SCHEMA IF EXISTS rowhints CASCADE;

CREATE SCHEMA rowhints;
SET search_path=rowhints;
SET optimizer_trace_fallback=on;

-- Setup tables
CREATE TABLE my_table(a int, b int);
CREATE INDEX my_awesome_index ON my_table(a);

CREATE TABLE your_table(a int, b int) WITH (appendonly=true);
CREATE INDEX your_awesome_index ON your_table(a);

CREATE TABLE our_table(a int, b int) PARTITION BY RANGE (a) (PARTITION p1 START(0) END(100) INCLUSIVE EVERY(20));
CREATE INDEX our_awesome_index ON our_table(a);

INSERT INTO my_table SELECT i, i FROM generate_series(1, 100)i;
INSERT INTO your_table SELECT i, i FROM generate_series(1, 100)i;
INSERT INTO our_table SELECT i, i FROM generate_series(1, 100)i;
ANALYZE my_table, your_table, our_table;

--------------------------------------------------------------------
-- Test the different row hint types:
--
--     - Absolute
--     - Add
--     - Subtract
--     - Multiply
--------------------------------------------------------------------

-- Baseline no hints
EXPLAIN SELECT t1.a, t2.a FROM my_table AS t1, your_table AS t2, our_table AS t3;

\o results/pg_hint_plan.tmpout
/*+
    Rows(t1 t2 t3 #123)
 */
EXPLAIN SELECT t1.a, t2.a FROM my_table AS t1, your_table AS t2, our_table AS t3;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout

\o results/pg_hint_plan.tmpout
/*+
    Rows(t1 t2 t3 +123)
 */
EXPLAIN SELECT t1.a, t2.a FROM my_table AS t1, your_table AS t2, our_table AS t3;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout

\o results/pg_hint_plan.tmpout
/*+
    Rows(t1 t2 t3 -123)
 */
EXPLAIN SELECT t1.a, t2.a FROM my_table AS t1, your_table AS t2, our_table AS t3;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout

\o results/pg_hint_plan.tmpout
/*+
    Rows(t1 t2 t3 *123)
 */
EXPLAIN SELECT t1.a, t2.a FROM my_table AS t1, your_table AS t2, our_table AS t3;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout


--------------------------------------------------------------------
--
-- Subqueries
--
--------------------------------------------------------------------

\o results/pg_hint_plan.tmpout
/*+
    Rows(my_table your_table #123)
 */
EXPLAIN SELECT * FROM my_table, (SELECT * FROM your_table) AS q;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout


--------------------------------------------------------------------
--
-- CTE
--
--------------------------------------------------------------------

\o results/pg_hint_plan.tmpout
/*+
    Rows(my_table your_table #123)
 */
EXPLAIN WITH cte AS (SELECT * FROM my_table, (SELECT * FROM your_table) as q) SELECT * FROM cte;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout


--------------------------------------------------------------------
-- Test updating lower join row hint
--------------------------------------------------------------------

-- force join order to isolate lower join row hint
set optimizer_join_order=query;

\o results/pg_hint_plan.tmpout
/*+
    Rows(t1 t2 #123)
 */
EXPLAIN SELECT t1.a, t2.a FROM my_table AS t1, your_table AS t2, our_table AS t3;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout

\o results/pg_hint_plan.tmpout
/*+
    Rows(t1 t2 *123)
 */
EXPLAIN SELECT t1.a, t2.a FROM my_table AS t1, your_table AS t2, our_table AS t3;
\o
\! sql/maskout.sh results/pg_hint_plan.tmpout
