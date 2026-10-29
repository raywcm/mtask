# Broker protocol v1

One Unix stream socket per task. Both peers verify UID. Hello must arrive within
two seconds and match the task and broker instance IDs.

## Frame

| Offset | Bytes | Field |
| --- | --- | --- |
| 0 | 2 | ASCII MT |
| 2 | 1 | Version: 1 |
| 3 | 1 | Message kind |
| 4 | 4 | Payload length, big-endian uint32 |
| 8 | N | Payload |

Input/Output carry raw bytes, at most 64 KiB. Other payloads are UTF-8 JSON or
empty, at most 8 KiB. Invalid or truncated frames close the connection. Writes
are serialized; a partly written frame is not retried.

| Kind | Message | Direction | Payload |
| --- | --- | --- | --- |
| 1 | Hello | Client → broker | task_id, instance_id, mode, optional size/attach_mode/capabilities |
| 2 | Welcome | Broker → client | IDs, replay, optional mode/size owner/size/capabilities |
| 3 | Input | Client → broker | Raw bytes; writable attach only |
| 4 | Output | Broker → client | Raw PTY bytes |
| 5 | Resize | Client → broker | rows/cols, nonzero uint16; size owner only |
| 6 | Detach | Client → broker | Empty; release connection |
| 7 | Exit | Broker → client | code, optional signal |
| 8 | Error | Broker → client | code, message; then close |
| 9 | Ping | Client → broker | Empty |
| 10 | Pong | Broker → client | Runtime JSON |
| 11 | Stop | Client → broker | Empty; idempotent request |
| 12 | Notice | Broker → client | role or refresh; negotiated clients only |

## Modes and compatibility

`mode` is control or attach. Control permits Ping/Stop/Detach and receives no
application output. Attach requires a size; `attach_mode` defaults to exclusive on the wire. New frontends negotiate shared
control by default, with exclusive fallback for brokers without mode support.
Preview clients set `preview: true` to avoid a redundant redraw request when
joining. Role notices may include the current `size`; older clients ignore it.
Takeover revokes controllers but keeps readers; shared adds a writer; readonly
rejects Input/Resize/Stop. The oldest connected controller owns the PTY size.

Non-default modes require `attach-modes-v1`. Clients first probe through control,
then request attach and verify the acknowledged mode. Older clients receive no
Notice frames. Unknown JSON fields allow optional extensions; invalid required
fields or a second JSON value are rejected.

EOF and Detach leave the task running. Stop returns Pong before completion;
clients may poll or await Exit. EOF without Exit does not determine task success.
At most 32 connections are allowed; idle control reads expire after 30 seconds.
Socket writes have a two-second deadline and slow subscribers are disconnected.

Reconnect uses `replay: none`. A newly created foreground task may receive
`initial`, with up to 2 MiB of startup output once. Recovery uses existing Resize
frames; screen snapshots and remote TCP are not part of v1. Disk schema and
protocol versions evolve independently.
