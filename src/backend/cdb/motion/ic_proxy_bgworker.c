/*-------------------------------------------------------------------------
 *
 * ic_proxy_bgworker.c
 *
 *    Interconnect Proxy Background Worker
 *
 * This is only a wrapper, the actual main loop is in ic_proxy_main.c .
 *
 *
 * Copyright (c) 2020-Present VMware, Inc. or its affiliates.
 *
 *
 *-------------------------------------------------------------------------
 */

#include "postgres.h"

#include "storage/ipc.h"
#include "storage/shmem.h"

#include "cdb/ic_proxy_bgworker.h"
#include "ic_proxy_server.h"

bool
ICProxyStartRule(Datum main_arg)
{
	return true;
}

/*
 * ICProxyMain
 */
void
ICProxyMain(Datum main_arg)
{
	/* main loop */
	proc_exit(ic_proxy_server_main());
}

/*
 * the size of ICProxy SHM structure
 */
Size
ICProxyShmemSize(void)
{
	Size		size = 0;
	size = add_size(size, sizeof(*ic_proxy_peer_listener_failed));
	return size;
}

/*
 * initialize ICProxy's SHM structure: only one flag variable
 */
void
ICProxyShmemInit(void)
{
	bool		found;
	ic_proxy_peer_listener_failed = ShmemInitStruct("IC_PROXY Listener Failure Flag",
													sizeof(*ic_proxy_peer_listener_failed),
													&found);
	if (!found)
		pg_atomic_init_u32(ic_proxy_peer_listener_failed, 0);
}