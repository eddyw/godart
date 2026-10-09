# Native descriptor and process contracts

This design defines acceptance requirements for native descriptor and process APIs.
File callbacks implement the ownership and readiness contract below.
Network ownership transfer and process inheritance remain incomplete.
The runtime code asset captures errno atomically for UDP connect, send, sendto, recvfrom and socket-name calls.
It also captures actual unlinkat, kill, read, write, fstatat and readiness probe errors within their syscall boundaries.
Other native calls still need atomic error boundaries.
Optimization and isolates can resume when a concrete parity blocker requires them.

## Source basis

The design uses Go 1.27.1 and the pinned Dart 3.13.4 SDK.
The original Go sources define ownership, callbacks, address families and launch errors:

- `src/syscall/net.go`
- `src/internal/poll/fd_unix.go`
- `src/net/rawconn.go`
- `src/net/file_unix.go`
- `src/net/fd_unix.go`
- `src/net/ipsock_posix.go`
- `src/net/sock_posix.go`
- `src/net/sockopt_linux.go` and `src/net/sockopt_bsd.go`
- `src/syscall/exec_unix.go`

Dart `lib/io/socket.dart` defines `ResourceHandle`, received control messages and `RawDatagramSocket.bind`.
The current runtime uses Dart-owned UDP sockets and borrows their actual descriptors.
That access does not transfer ownership or permit a second close.

## Descriptor ownership

A native descriptor needs an owner before the runtime exposes it.
The owner retains the descriptor, open/closed state, pending operations and deadline state.
All callbacks and operations retain that owner until their native access ends.
Close rejects new operations and cancels pending operations.
The runtime closes the descriptor only after active access ends.
A later descriptor reuse must not satisfy an earlier pending operation.

`net.FileConn`, `net.FileListener` and `net.FilePacketConn` duplicate the caller's descriptor.
The caller retains its file. The new network object owns the duplicate.
The duplicate uses close-on-exec and nonblocking mode, as the original Go adapter requires.
`TCPConn.File`, `TCPListener.File` and corresponding APIs return a separately owned duplicate.
Closure of either owner must preserve the other owner's valid descriptor.
Duplication can share native file status flags. Descriptor ownership does not imply independent native flags.

An integer alone cannot establish descriptor type, ownership or readiness.
Native metadata and socket queries must establish the descriptor contract before adoption.
A failed adoption closes only the new duplicate.
No public API can return a fabricated descriptor or a successful placeholder.

## RawConn and readiness

`os.File.SyscallConn` supports owned POSIX file descriptors on Linux and macOS.
Its callbacks execute as Go callbacks and can suspend or panic.
The file retains the actual descriptor until callback access ends.
Pollable Close cancels pending probes and waits for active callbacks.
Blocking file Close can return before a callback, but actual closure waits for the final reference.
Standard stream Close retains its existing no-op behavior. Pipe Fd retains nonblocking mode.
These limits differ from Go. Web targets return `ENOSYS`.
Linux and macOS TCP listeners own native descriptors through the C bridge.
`TCPListener.File` duplicates the descriptor with close-on-exec.
`net.FileListener` duplicates a TCP source descriptor and preserves caller ownership.
Each owner can close independently. The accepted TCP connection retains its descriptor after listener close.
The native stream preserves read and write deadlines, partial writes, half close and socket options.
Dialed TCP connections and Unix listeners still use Dart socket ownership.
TCPListener.SyscallConn retains the native descriptor for the complete Control callback.
Close waits for active callbacks. Callback panic releases ownership.
Listener RawConn Read and Write return EINVAL without a callback.
Unix listener descriptors and connection descriptor APIs remain incomplete.

`Control` retains the descriptor for the complete callback.
Its guarantee ends when the callback returns.
The runtime must preserve operation errors, callback panics and the original `net.OpError` metadata.

`Read` and `Write` serialize operations in their respective directions.
They prepare the poll state before the first callback, as the original Go implementation does.
A prior close or deadline can return an error before that callback.
A true callback result completes the operation.
A false result suspends the goroutine until actual readiness, closure or a deadline permits another action.
The runtime must retain callback validity and descriptor ownership across that suspension.
Readiness does not guarantee that the next syscall succeeds.
Another operation can consume the available data before that syscall.

The initial native design uses nonblocking descriptors and actual zero-timeout readiness probes.
A scheduler timer requests another probe after an unready result.
The timer itself never establishes readiness.
This design avoids a native wait that blocks the single Dart isolate.
It does not claim native Go poller performance.
Network callback suspension requires the same compiler and runtime contract as file callbacks.
A synchronous callback wrapper must not silently replace a permitted Go callback contract.

Acceptance must cover:

- Immediate callback completion and repeated false callback results.
- Actual EAGAIN, partial reads, partial writes and empty datagrams.
- Concurrent same-direction operations and independent read/write progress.
- Close during a pending operation and close during a callback.
- Deadline changes, deadline cancellation and descriptor reuse after close.
- Adoption failure, duplicate ownership and actual descriptor exhaustion.
- Original listener restrictions on RawConn read and write operations.

The existing separate errno read cannot establish atomic syscall/error capture.
That limitation remains an explicit acceptance blocker for exact native error behavior.

## Process launch and inheritance

A launch operation must distinguish child setup failure from successful execution followed by an exit.
A child exit of 127 does not establish ENOEXEC or any other launch error.
Error text and signal status cannot supply that distinction.

The required native boundary creates a close-on-exec child status pipe before the launch.
The child writes its actual setup or exec error through that pipe.
Successful exec closes the write end through close-on-exec.
The parent reads the complete error value or EOF, as the original status-pipe contract requires.
EOF does not prove that the child program executed user code.
It closes both pipe ends on their respective ownership paths.
A failed launch requires child reap and cleanup before the parent returns.
No Dart code may execute in the child between fork and exec.
The Dart VM can have native threads even when Go goroutines use 1 isolate.

The same boundary must implement the original descriptor map and preserve source descriptors.
It must handle descriptor cycles, nil file entries, close-on-exec and unrelated inherited descriptors.
It must apply the requested directory, environment, signal state and supported process attributes before exec.
An unsupported process attribute must produce an explicit error.
A process proxy or shell cannot establish this contract.

Acceptance must distinguish:

- A valid executable that returns 127.
- A missing executable, a denied executable and an invalid executable format.
- A failed child directory or descriptor setup.
- Successful stdio and ExtraFiles inheritance, including descriptor cycles.
- Actual child signal dispositions and descriptor leak prevention.
- Launch failure cleanup and successful child wait results.

Linux and macOS implement the child error pipe and parent-owned native stdio through the existing C code asset.
The child uses only async-signal-safe operations until `execve`.
The bridge preserves actual launch errors, raw argument bytes, environment bytes, relative paths and child wait results.
It distinguishes valid exit 127 from invalid format and retains cleanup for failed launches.
Native child signals use the Go platform number, rather than a Dart enum number.

Dart's child reaper can consume an unregistered native child status on Linux and macOS.
The bridge first uses `waitpid` and retains a real Linux pidfd when the kernel supplies it.
On `ECHILD`, `PIDFD_GET_INFO` can provide the actual retained exit status.
The bridge uses that value only when the kernel returns the exit-status mask bit.
Linux 7.0 defines this public metadata contract in [`include/uapi/linux/pidfd.h`](https://github.com/torvalds/linux/blob/v7.0/include/uapi/linux/pidfd.h).
Older kernels cannot recover that status after another reaper consumes it.
Their wait retains the actual `ECHILD` error, without an inferred exit value.

On macOS, the parent registers the child PID with an owned kqueue before the child executes.
The child cannot exit before native status registration.
After `ECHILD`, the bridge reads the actual captured status only when `NOTE_EXITSTATUS` is present.
Apple defines this public contract in [`bsd/sys/event.h`](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/event.h).
The parent closes the status descriptor after wait or launch cleanup.

Native POSIX launch preserves the child name from `argv[0]`.
The name can be empty or contain non-UTF-8 bytes.
`ExtraFiles`, `SysProcAttr` and full descriptor-map inheritance remain incomplete.
Native POSIX children use the current Go SIGPIPE Ignore state.
Notify for SIGPIPE restores the default child disposition. Reset retains an explicit Ignore, as native Go does.
The bridge retains other inherited Dart dispositions and the signal mask.
External startup dispositions and SIGTTOU still have host limits.
These limits do not establish complete process parity.

## IPv6 and UDP creation

The socket creator must choose the native family before bind.
The original Go family rules distinguish `udp`, `udp4` and `udp6`.
`udp6` requires AF_INET6 and the original IPv6-only option attempt before bind.
A wildcard `udp` listener can require an IPv6 socket with IPv4 mapping.
The implementation must preserve original platform-specific option error policy.
Go's Linux and BSD default adapters ignore the IPv6-only option error.
It still observes errors from required broadcast options.

Dart `RawDatagramSocket.bind` exposes reuseAddress, reusePort and ttl.
It exposes no IPv6-only parameter.
A descriptor borrowed after Dart bind cannot establish the required pre-bind behavior.
The native creator must own a real socket, apply the original options and then bind or connect it.
It must retain address bytes, port, scope ID, local address and peer address from actual native operations.

The descriptor owner and readiness contract above apply to these sockets.
The first implementation must retain native ownership through its own read/write adapter.
It must not manufacture a Dart ResourceHandle from locally assembled control bytes.
Dart permits `extractHandles` only on actual received control messages and only 1 time.
A future Dart ownership transfer would need a real kernel descriptor transfer and separate acceptance.

Acceptance must cover:

- IPv4, IPv6-only and wildcard dual-family bind and dial behavior.
- IPv4-mapped addresses, explicit zones and actual interface scope IDs.
- Incompatible local/remote families and exact typed errors.
- Empty datagrams, actual sender addresses, message flags and control bytes.
- Deadline changes, cancellation, close ownership and descriptor exhaustion.
- Original socket-control callback order before bind or connect.

UDP bridge calls return their actual errno within the native result.
Other native calls still need the same atomic boundary.
The current Dart-owned UDP adapter retains its scoped acceptance.
It does not establish full IPv6, RawConn or descriptor inheritance parity.
