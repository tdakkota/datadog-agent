//+build ignore

package ebpf

/*
#include "./c/runtime/conntrack-types.h"
*/
import "C"

type In6Addr C.struct_in6_addr

type ConntrackTuple C.conntrack_tuple_t

type ConntrackTelemetry C.conntrack_telemetry_t
