# System Catalog Functions

Greenplum Database provides the following system catalog functions:

> **Note** This list is provisional and may be incomplete.

-   [gp_get_suboverflowed_backends](#gp_get_suboverflowed_backends)
-   [pg_stat_get_backend_subxact](#pg_stat_get_backend_subxact)

## <a id="statistics"></a>Statistics Functions

The following functions support collection and reporting of information about server activity.

### <a id="gp_get_suboverflowed_backends"></a>gp_get_suboverflowed_backends

`gp_get_suboverflowed_backends()` return type is `integer[]`.

Returns an array of integers which indicate the process IDs  of all the sessions in which a backend has subtransaction overflows on the running segment. For a cluster-wide view of the suboverflowed backends on every segment, use the view [gp_suboverflowed_backend](catalog_ref-views.html#gp_suboverflowed_backend).

### <a id="pg_stat_get_backend_subxact"></a>pg_stat_get_backend_subxacts

`pg_stat_get_backend_subxact(integer)` return type is `record`.

Returns a record of information about the subtransactions of the backend with the specified ID. The fields returned are `subxact_count`, which is the number of subtransactions in the backend's subtransaction cache, and `subxact_overflow`, which indicates whether the backend's subtransaction cache is overflowed or not.
