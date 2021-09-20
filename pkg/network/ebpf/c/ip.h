#ifndef __IP_H
#define __IP_H

#include <linux/kconfig.h>

#include "bpf_helpers.h"
#include "bpf_endian.h"

#include <uapi/linux/if_ether.h>
#include <uapi/linux/in.h>
#include <uapi/linux/ip.h>
#include <uapi/linux/ipv6.h>
#include <uapi/linux/tcp.h>
#include <uapi/linux/udp.h>

#include <net/ipv6.h>

static __always_inline bool is_ipv4_set(const struct in6_addr *ip) {
    return ip->s6_addr32[3] != 0;
}

static __always_inline void set_ipv4(struct in6_addr *dst, const __be32 src) {
    dst->s6_addr32[3] = src;
}

static __always_inline bool is_ipv6_set(const struct in6_addr *ip) {
    return (ip->s6_addr32[0] | ip->s6_addr32[1] | ip->s6_addr32[2] | ip->s6_addr32[3]) != 0;
}

static __always_inline void read_ipv6_skb(struct __sk_buff *skb, __u64 off, struct in6_addr *addr) {
    ipv6_addr_set(addr,
        bpf_ntohl(load_word(skb, off)),
        bpf_ntohl(load_word(skb, off + 4)),
        bpf_ntohl(load_word(skb, off + 8)),
        bpf_ntohl(load_word(skb, off + 12))
    );
}

static __always_inline void read_ipv4_skb_offset(struct in6_addr *dst, struct __sk_buff *skb, __u64 offset) {
    set_ipv4(dst, bpf_ntohl(load_word(skb, offset)));
}

static __always_inline int read_ipv4(struct in6_addr *dst, __be32 *src) {
    return bpf_probe_read(&dst->s6_addr32[3], sizeof(__be32), src);
}

static __always_inline int read_ipv4_sock_offset(struct in6_addr *dst, struct sock *skp, __u64 offset) {
    return bpf_probe_read(&dst->s6_addr32[3], sizeof(__be32), ((char*)skp) + offset);
}

static __always_inline int read_ipv4_flow_offset(struct in6_addr *dst, const struct flowi4 *fl4, __u64 offset) {
    return bpf_probe_read(&dst->s6_addr32[3], sizeof(__be32), ((char*)fl4) + offset);
}

static __always_inline int read_in_addr(struct in6_addr *dst, const struct in_addr *src) {
    return bpf_probe_read(&dst->s6_addr32[3], sizeof(__be32), (void *)&src->s_addr);
}

static __always_inline __u64 read_conn_tuple_skb(struct __sk_buff *skb, skb_info_t *info) {
    __builtin_memset(info, 0, sizeof(skb_info_t));
    info->data_off = ETH_HLEN;

    __u16 l3_proto = load_half(skb, offsetof(struct ethhdr, h_proto));
    __u8 l4_proto = 0;
    switch (l3_proto) {
    case ETH_P_IP:
    {
        __u8 ipv4_hdr_len = (load_byte(skb, info->data_off) & 0x0f) << 2;
        if (ipv4_hdr_len < sizeof(struct iphdr)) {
            return 0;
        }
        l4_proto = load_byte(skb, info->data_off + offsetof(struct iphdr, protocol));
        info->tup.metadata |= CONN_V4;
        read_ipv4_skb_offset(&info->tup.saddr, skb, info->data_off + offsetof(struct iphdr, saddr));
        read_ipv4_skb_offset(&info->tup.daddr, skb, info->data_off + offsetof(struct iphdr, daddr));
        info->data_off += ipv4_hdr_len;
        break;
    }
    case ETH_P_IPV6:
        l4_proto = load_byte(skb, info->data_off + offsetof(struct ipv6hdr, nexthdr));
        info->tup.metadata |= CONN_V6;
        read_ipv6_skb(skb, info->data_off + offsetof(struct ipv6hdr, saddr), &info->tup.saddr);
        read_ipv6_skb(skb, info->data_off + offsetof(struct ipv6hdr, daddr), &info->tup.daddr);
        info->data_off += sizeof(struct ipv6hdr);
        break;
    default:
        return 0;
    }

    switch (l4_proto) {
    case IPPROTO_UDP:
        info->tup.metadata |= CONN_TYPE_UDP;
        info->tup.sport = load_half(skb, info->data_off + offsetof(struct udphdr, source));
        info->tup.dport = load_half(skb, info->data_off + offsetof(struct udphdr, dest));
        info->data_off += sizeof(struct udphdr);
        break;
    case IPPROTO_TCP:
        info->tup.metadata |= CONN_TYPE_TCP;
        info->tup.sport = load_half(skb, info->data_off + offsetof(struct tcphdr, source));
        info->tup.dport = load_half(skb, info->data_off + offsetof(struct tcphdr, dest));

        info->tcp_flags = load_byte(skb, info->data_off + TCP_FLAGS_OFFSET);
        // TODO: Improve readability and explain the bit twiddling below
        info->data_off += ((load_byte(skb, info->data_off + offsetof(struct tcphdr, ack_seq) + 4) & 0xF0) >> 4) * 4;
        break;
    default:
        return 0;
    }

    if ((skb->len - info->data_off) < 0) {
        return 0;
    }

    return 1;
}

static __always_inline void flip_tuple(conn_tuple_t *t) {
    // TODO: we can probably replace this by swap operations
    __u16 tmp_port = t->sport;
    t->sport = t->dport;
    t->dport = tmp_port;

    struct in6_addr tmp_ip = t->saddr;
    t->saddr = t->daddr;
    t->daddr = tmp_ip;
}

static __always_inline void print_ip(struct in6_addr ip, u16 port, u32 metadata) {
    if (metadata & CONN_V6) {
        log_debug("v6 %llx%llx:%u\n", bpf_ntohll(*(__be64*)&ip.s6_addr32[0]), bpf_ntohll(*(__be64*)&ip.s6_addr32[2]), port);
    } else {
        log_debug("v4 %x:%u\n", bpf_ntohl(ip.s6_addr32[3]), port);
    }
}

#endif
