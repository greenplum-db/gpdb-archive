-- @Description Tests basic index stats after vacuuming

CREATE TABLE mytab(
          col_int int,
          col_text text,
          col_numeric numeric,
          col_unq int
          ) with(appendonly=true) DISTRIBUTED RANDOMLY;

Create index mytab_int_idx1 on mytab(col_int);

insert into mytab values(1,'aa',1001,101),(2,'bb',1002,102);

select * from mytab;
update mytab set col_text=' new value' where col_int = 1;
select * from mytab;
vacuum mytab;
SELECT relname, reltuples FROM pg_class WHERE relname = 'mytab';
SELECT relname, reltuples FROM pg_class WHERE relname = 'mytab_int_idx1';

-- A test of index stat for access methods that rely on table tuple count (bitmap, gin)
truncate mytab;
create index mytab_int_idx2 on mytab using bitmap(col_int);

insert into mytab values(1,'aa',1001,101),(2,'bb',1002,102);

SELECT relname, reltuples FROM pg_class WHERE relname = 'mytab_int_idx2';

-- first vacuum collect table stat on segments
vacuum mytab;
-- second vacuum update index stat with table stat
vacuum mytab;

SELECT relname, reltuples FROM pg_class WHERE relname = 'mytab_int_idx2';
