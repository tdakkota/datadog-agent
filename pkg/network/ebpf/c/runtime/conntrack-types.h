#ifndef __CONNTRACK_TYPES_H
#define __CONNTRACK_TYPES_H

#include <linux/types.h>
#include <linux/in6.h>

typedef struct {
    struct in6_addr saddr;
    struct in6_addr daddr;
    __u16 sport;
    __u16 dport;
    __u32 netns;
    // Metadata description:
    // First bit indicates if the connection is TCP (1) or UDP (0)
    // Second bit indicates if the connection is V6 (1) or V4 (0)
    __u32 metadata; // This is that big because it seems that we atleast need a 32-bit aligned struct

    __u32 _pad;
} conntrack_tuple_t;

typedef struct {
    __u64 registers;
    __u64 registers_dropped;
} conntrack_telemetry_t;

enum conntrack_telemetry_counter {
    registers,
    registers_dropped,
};

#endif
