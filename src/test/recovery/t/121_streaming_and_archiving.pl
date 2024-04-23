use strict;
use warnings;
use PostgreSQL::Test::Cluster;
use PostgreSQL::Test::Utils;
use Test::More;

my $node_primary;
my $node_standby;

##################### Test with wal_sender_archiving_status_interval disabled #####################

# Initialize primary node with WAL archiving setup and wal_sender_archiving_status_interval as disabled
$node_primary = PostgreSQL::Test::Cluster->new('primary');
$node_primary->init(
	has_archiving    => 1,
	allows_streaming => 1);
$node_primary->append_conf('postgresql.conf', 'wal_sender_archiving_status_interval = 0');
$node_primary->start;
my $primary_data = $node_primary->data_dir;

# Set an incorrect archive_command so that archiving fails
my $incorrect_command = "exit 1";
$node_primary->safe_psql(
'postgres', qq{
ALTER SYSTEM SET archive_command TO '$incorrect_command';
SELECT pg_reload_conf();
});

# Take backup
$node_primary->backup('my_backup');

# Initialize the standby node. It will inherit wal_sender_archiving_status_interval from the primary
$node_standby = PostgreSQL::Test::Cluster->new('standby');
$node_standby->init_from_backup($node_primary, 'my_backup', has_streaming => 1);
my $standby_data = $node_standby->data_dir;
$node_standby->start;

$node_primary->safe_psql(
	'postgres', q{
	CREATE TABLE test1 AS SELECT generate_series(1,10) AS x;
});

my $current_walfile = $node_primary->safe_psql('postgres', "SELECT pg_walfile_name(pg_current_wal_lsn())");
$node_primary->safe_psql(
	'postgres', q{
	SELECT pg_switch_wal();
	CHECKPOINT;
});

# After switching wal, the current wal file will be marked as ready to be archived on the primary. But this wal file
# won't get archived because of the incorrect archive_command
my $walfile_ready = "pg_wal/archive_status/$current_walfile.ready";
my $walfile_done = "pg_wal/archive_status/$current_walfile.done";

# Wait for the standby to catch up
$node_primary->wait_for_catchup($node_standby, 'replay',
	$node_primary->lsn('insert'));

# Wait for archive failure
$node_primary->poll_query_until('postgres',
q{SELECT failed_count > 0 FROM pg_stat_archiver}, 't')
or die "Timed out while waiting for archiving to fail";

# Due to archival failure, check if primary has only .ready wal file and not .done
# Check if standby has .done created as wal_sender_archiving_status_interval is disabled.
ok( -f "$primary_data/$walfile_ready",
	".ready file exists on primary for WAL segment $current_walfile"
);
ok( !-f "$primary_data/$walfile_done",
	".done file does not exist on primary for WAL segment $current_walfile"
);

ok( !-f "$standby_data/$walfile_ready",
	".ready file does not exist on standby for WAL segment $current_walfile when wal_sender_archiving_status_interval=0"
);
ok( -f "$standby_data/$walfile_done",
	".done file exists on standby for WAL segment $current_walfile when wal_sender_archiving_status_interval=0"
);

##################### Test with wal_sender_archiving_status_interval enabled #####################
$node_primary->adjust_conf('postgresql.conf', 'wal_sender_archiving_status_interval', "50ms");
$node_primary->reload;
# We need to enable wal_sender_archiving_status_interval on the standby as well. Technically the value doesn't matter
# since the wal_receiver process only cares about this guc being set and not the actual value.
$node_standby->adjust_conf('postgresql.conf', 'wal_sender_archiving_status_interval', "50ms");
$node_standby->reload;

$node_primary->safe_psql(
	'postgres', q{
	CREATE TABLE test2 AS SELECT generate_series(1,10) AS x;
});

my $current_walfile2 = $node_primary->safe_psql('postgres', "SELECT pg_walfile_name(pg_current_wal_lsn())");
my $walfile_ready2 = "pg_wal/archive_status/$current_walfile2.ready";
my $walfile_done2 = "pg_wal/archive_status/$current_walfile2.done";

$node_primary->safe_psql(
	'postgres', q{
	SELECT pg_switch_wal();
	CHECKPOINT;
});

# Wait for standby to catch up
$node_primary->wait_for_catchup($node_standby, 'replay',
	$node_primary->lsn('insert'));

# Check if primary and standby have created .ready file for the wal that failed to archive
ok( -f "$primary_data/$walfile_ready2",
	".ready file exists on primary for WAL segment $current_walfile2 when wal_sender_archiving_status_interval=50ms"
);
ok( -f "$standby_data/$walfile_ready2",
	".ready file exists on standby for WAL segment $current_walfile2 when wal_sender_archiving_status_interval=50ms"
);

ok( !-f "$primary_data/$walfile_done2",
	".done file does not exist on primary for WAL segment $current_walfile2 when wal_sender_archiving_status_interval=50ms"
);
ok( !-f "$standby_data/$walfile_done2",
	".done file does not exist on standby for WAL segment $current_walfile2 when wal_sender_archiving_status_interval=50ms"
);

# Make WAL archiving work again for primary by resetting the archive_command
$node_primary->safe_psql(
	'postgres', q{
	ALTER SYSTEM RESET archive_command;
	SELECT pg_reload_conf();
});

# Wait until all the failed files get archived by the primary
$node_primary->poll_query_until('postgres',"SELECT archived_count=failed_count FROM pg_stat_archiver", 't')
  or die "Timed out while waiting for archiving to finish";

# Check if primary has .done file created for the archived segment and also that the file gets uploaded to the archive
ok( -f "$primary_data/$walfile_done2",
	".done file exists for WAL segment $current_walfile2");
ok( !-f "$primary_data/$walfile_ready2",
	".ready file does not exist for WAL segment $current_walfile2");
ok( -f $node_primary->archive_dir . "/$current_walfile2",
	"$current_walfile2 exists in primary's archive");

# Check if standby has .done file created for the archived segment
wait_until_file_exists($node_standby, "$standby_data/$walfile_done2",
	".done file to exist on standby for WAL segment $current_walfile2"
);
ok( !-f "$standby_data/$walfile_ready2",
	".ready file does not exist on standby for WAL segment $current_walfile2");

ok( !-f $node_standby->archive_dir . "/$current_walfile2",
	"$current_walfile2 does not exist in standby's archive");

################### Now again make archiving fail but this time promote the standby and let the standby archive the wal
$node_primary->safe_psql(
	'postgres', qq{
	ALTER SYSTEM SET archive_command TO '$incorrect_command';
	SELECT pg_reload_conf();
});
# make archiving work for the standby node
$node_standby->safe_psql(
	'postgres', q{
	ALTER SYSTEM RESET archive_command;
	SELECT pg_reload_conf();
});

$node_primary->safe_psql(
	'postgres', q{
	CREATE TABLE test3 AS SELECT generate_series(1,10) AS x;
});
my $current_walfile3 = $node_primary->safe_psql('postgres', "SELECT pg_walfile_name(pg_current_wal_lsn())");
$node_primary->safe_psql(
	'postgres', q{
	SELECT pg_switch_wal();
	CHECKPOINT;
});
my $walfile_ready3 = "pg_wal/archive_status/$current_walfile3.ready";
my $walfile_done3 = "pg_wal/archive_status/$current_walfile3.done";

# Wait for the standby to catch up
$node_primary->wait_for_catchup($node_standby, 'replay',
	$node_primary->lsn('insert'));

ok( -f "$standby_data/$walfile_ready3",
	".ready file exists on standby for WAL segment $current_walfile3 when wal_sender_archiving_status_interval=50ms"
);
ok( !-f "$standby_data/$walfile_done3",
	".done file does not exist on standby for WAL segment $current_walfile3 when wal_sender_archiving_status_interval=50ms"
);
ok( !-f $node_primary->archive_dir . "/$current_walfile3",
	"$current_walfile3 does not exist in the archive");

# Now promote the standby
$node_standby->promote;

# Wait for promotion to complete
$node_standby->poll_query_until('postgres', "SELECT NOT pg_is_in_recovery();")
or die "Timed out while waiting for promotion";

# Wait until the wal file gets archived by the standby, Note that the archive dir is primary's archive dir and not standby's.
# This is because archive_command (which has the cp command) gets copied over from the primary node when we initialize the standby
wait_until_file_exists($node_primary, $node_primary->archive_dir . "/$current_walfile3",
	"$current_walfile3 to be archived by the standby");

ok( -f "$primary_data/$walfile_ready3",
	".ready file exists on primary for WAL segment $current_walfile3 when wal_sender_archiving_status_interval=50ms"
);
# primary won't be able to archive and hence won't have the .done file
ok( !-f "$primary_data/$walfile_done3",
	".done file does not exist on primary for WAL segment $current_walfile3 when wal_sender_archiving_status_interval=50ms"
);
wait_until_file_exists($node_standby, "$standby_data/$walfile_done3",
	".done file to exist on standby for WAL segment $current_walfile3 when wal_sender_archiving_status_interval=50ms"
);
ok( !-f "$standby_data/$walfile_ready3",
	".ready file does not exist on standby for WAL segment $current_walfile3 when wal_sender_archiving_status_interval=50ms"
);

done_testing();

