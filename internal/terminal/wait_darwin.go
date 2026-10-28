package terminal

import "golang.org/x/sys/unix"

func waitTTY(fd int, event int16, milliseconds int) (bool, error) {
	// Darwin poll(/dev/tty) returns POLLNVAL for a valid controlling TTY.
	// Reading anyway returns EAGAIN and spins. select supports this device.
	if fd < 0 || fd >= unix.FD_SETSIZE {
		return false, unix.EBADF
	}
	var fds unix.FdSet
	fds.Set(fd)
	var read, write *unix.FdSet
	if event == unix.POLLIN {
		read = &fds
	} else {
		write = &fds
	}
	timeout := unix.NsecToTimeval(int64(milliseconds) * 1_000_000)
	n, err := unix.Select(fd+1, read, write, nil, &timeout)
	return n > 0, err
}
