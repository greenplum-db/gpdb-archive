/*-------------------------------------------------------------------------
 *
 * ic_proxy_bgworker.h
 *	  TODO file description
 *
 *
 * Copyright (c) 2020-Present VMware, Inc. or its affiliates.
 *
 *
 *-------------------------------------------------------------------------
 */

#ifndef IC_PROXY_BGWORKER_H
#define IC_PROXY_BGWORKER_H

#include "port/atomics.h"

/* flag (in SHM) for incidaing if peer listener bind/listen failed */
extern pg_atomic_uint32 *ic_proxy_peer_listener_failed;

extern bool ICProxyStartRule(Datum main_arg);
extern void ICProxyMain(Datum main_arg);
extern Size ICProxyShmemSize(void);
extern void ICProxyShmemInit(void);

#endif   /* IC_PROXY_BGWORKER_H */
