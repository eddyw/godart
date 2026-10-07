import 'dart:ffi';

final class NativeResult extends Struct {
  @IntPtr()
  external int value;

  @Int()
  external int error;
}

@Native<NativeResult Function(Int, Pointer<Void>, Uint32)>(
  symbol: 'goruntime_connect',
)
external NativeResult nativeConnect(int fd, Pointer<Void> address, int length);

@Native<NativeResult Function(Int, Pointer<Void>, Size, Int)>(
  symbol: 'goruntime_send',
)
external NativeResult nativeSend(
  int fd,
  Pointer<Void> data,
  int length,
  int flags,
);

@Native<
  NativeResult Function(Int, Pointer<Void>, Size, Int, Pointer<Void>, Uint32)
>(symbol: 'goruntime_sendto')
external NativeResult nativeSendto(
  int fd,
  Pointer<Void> data,
  int length,
  int flags,
  Pointer<Void> address,
  int addressLength,
);

@Native<
  NativeResult Function(
    Int,
    Pointer<Void>,
    Size,
    Int,
    Pointer<Void>,
    Pointer<Uint32>,
  )
>(symbol: 'goruntime_recvfrom')
external NativeResult nativeRecvfrom(
  int fd,
  Pointer<Void> data,
  int length,
  int flags,
  Pointer<Void> address,
  Pointer<Uint32> addressLength,
);

@Native<NativeResult Function(Int, Pointer<Void>, Pointer<Uint32>)>(
  symbol: 'goruntime_getsockname',
)
external NativeResult nativeGetsockname(
  int fd,
  Pointer<Void> address,
  Pointer<Uint32> length,
);

@Native<NativeResult Function(Int, Pointer<Void>, Pointer<Uint32>)>(
  symbol: 'goruntime_getpeername',
)
external NativeResult nativeGetpeername(
  int fd,
  Pointer<Void> address,
  Pointer<Uint32> length,
);

@Native<NativeResult Function(Int, Pointer<Uint8>, Int)>(
  symbol: 'goruntime_unlinkat',
)
external NativeResult nativeUnlinkat(int fd, Pointer<Uint8> path, int flags);

@Native<NativeResult Function(Int, Int)>(symbol: 'goruntime_kill')
external NativeResult nativeKill(int pid, int signal);

@Native<NativeResult Function(Int, Pointer<Void>, Size)>(
  symbol: 'goruntime_read',
)
external NativeResult nativeRead(int fd, Pointer<Void> data, int length);

@Native<NativeResult Function(Int, Pointer<Void>, Size)>(
  symbol: 'goruntime_write',
)
external NativeResult nativeWrite(int fd, Pointer<Void> data, int length);

@Native<NativeResult Function(Int, Int)>(symbol: 'goruntime_poll')
external NativeResult nativePoll(int fd, int writeReady);

final class NativeProcessResult extends Struct {
  @Int()
  external int pid;
  @Int()
  external int error;
  @Int()
  external int input;
  @Int()
  external int output;
  @Int()
  external int diagnostic;
  @Int()
  external int launch;
  @Int()
  external int statusfd;
}

final class NativeProcessState extends Struct {
  @Int()
  external int value;
  @Int()
  external int error;
  @Int()
  external int complete;
}

@Native<
  NativeProcessResult Function(
    Pointer<Uint8>,
    Pointer<Pointer<Uint8>>,
    Pointer<Pointer<Uint8>>,
    Pointer<Uint8>,
  )
>(symbol: 'goruntime_process_start')
external NativeProcessResult nativeProcessStart(
  Pointer<Uint8> path,
  Pointer<Pointer<Uint8>> argv,
  Pointer<Pointer<Uint8>> environment,
  Pointer<Uint8> directory,
);

@Native<NativeProcessState Function(Int, Int)>(symbol: 'goruntime_process_wait')
external NativeProcessState nativeProcessWait(int pid, int statusfd);

@Native<NativeResult Function(Int, Int, Int)>(symbol: 'goruntime_process_kill')
external NativeResult nativeProcessKill(int pid, int statusfd, int signal);

@Native<NativeResult Function(Int, Pointer<Uint8>, Pointer<Void>, Int)>(
  symbol: 'goruntime_fstatat',
)
external NativeResult nativeFstatat(
  int fd,
  Pointer<Uint8> path,
  Pointer<Void> data,
  int flags,
);
