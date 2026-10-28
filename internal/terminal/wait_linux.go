package terminal

import "golang.org/x/sys/unix"

func waitTTY(fd int, event int16, milliseconds int) (bool, error) {
	if fd < 0 {
		return false, unix.EBADF
	}
	fds := []unix.PollFd{{Fd: int32(fd), Events: event}}
	n, err := unix.Poll(fds, milliseconds)
	if err != nil || n == 0 {
		return false, err
	}
	if fds[0].Revents&unix.POLLNVAL != 0 {
		return false, unix.EBADF
	}
	if fds[0].Revents&unix.POLLERR != 0 && fds[0].Revents&event == 0 {
		return false, unix.EIO
	}
	return fds[0].Revents&(event|unix.POLLHUP) != 0, nil
}
