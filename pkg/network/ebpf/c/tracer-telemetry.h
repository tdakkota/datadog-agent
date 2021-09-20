#ifndef __TRACER_TELEMETRY_H
#define __TRACER_TELEMETRY_H

#include "tracer-maps.h"
#include "bpf_endian.h"
#include "ip.h"
#include "ipv6.h"

#include <linux/kconfig.h>
#include <net/sock.h>
#include <net/ipv6.h>

enum telemetry_counter
{
    missed_tcp_close,
    missed_udp_close,
    udp_send_processed,
    udp_send_missed,
    conn_stats_max_entries_hit,
};

static __always_inline void increment_telemetry_count(enum telemetry_counter counter_name) {
    __u64 key = 0;
    telemetry_t *val = NULL;
    val = bpf_map_lookup_elem(&telemetry, &key);
    if (val == NULL) {
        return;
    }

    switch (counter_name) {
    case missed_tcp_close:
        __sync_fetch_and_add(&val->missed_tcp_close, 1);
        break;
    case missed_udp_close:
        __sync_fetch_and_add(&val->missed_udp_close, 1);
        break;
    case udp_send_processed:
        __sync_fetch_and_add(&val->udp_sends_processed, 1);
        break;
    case udp_send_missed:
        __sync_fetch_and_add(&val->udp_sends_missed, 1);
        break;
    case conn_stats_max_entries_hit:
        __sync_fetch_and_add(&val->conn_stats_max_entries_hit, 1);
        break;
    }
}

static __always_inline void sockaddr_to_addr(struct sockaddr *sa, struct in6_addr *addr, u16 *port) {
    if (!sa) {
        return;
    }

    u16 family = 0;
    bpf_probe_read(&family, sizeof(family), &sa->sa_family);

    struct sockaddr_in *sin;
    struct sockaddr_in6 *sin6;
    switch (family) {
    case AF_INET:
        sin = (struct sockaddr_in *)sa;
        if (addr) {
            read_in_addr(addr, &sin->sin_addr);
        }
        if (port) {
            bpf_probe_read(port, sizeof(__be16), &sin->sin_port);
            *port = bpf_ntohs(*port);
        }
        break;
    case AF_INET6:
        sin6 = (struct sockaddr_in6 *)sa;
        if (addr) {
            read_in6_addr(addr, &sin6->sin6_addr);
        }
        if (port) {
            bpf_probe_read(port, sizeof(u16), &sin6->sin6_port);
            *port = bpf_ntohs(*port);
        }
        break;
    }
}

#endif // __TRACER_TELEMETRY_H
