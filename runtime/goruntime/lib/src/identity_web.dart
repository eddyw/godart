import 'slice.dart';

int $getuid() => -1;
int $geteuid() => -1;
int $getgid() => -1;
int $getegid() => -1;
int $getppid() => 0;
int $getpgrp() => -1;
(int, int) $getpgid(int pid) => (-1, -1);
(int, int) $getsid(int pid) => (-1, -1);
int $getpagesize() => 4096;

(Slice<int>, int) $getgroups() => (Slice<int>.nil(), -1);

int $sendSignal(int pid, int signal) => -1;
