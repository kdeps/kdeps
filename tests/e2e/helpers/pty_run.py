"""Run a command in a pseudo-terminal and type lines into it.

    python3 pty_run.py <transcript> <startup secs> <line>... -- <command>...

Waits for the command to draw, types every line at once (each ends with Enter),
then waits for it to exit. Answers the terminal queries a full-screen program
sends at startup. Writes everything the command drew to <transcript> and exits
with the command's status. Exits 3 where pty is unavailable (Windows).
"""
import os, select, sys, time

try:
    import fcntl, pty, struct, termios
except ImportError:
    sys.exit(3)

out_path, startup = sys.argv[1], float(sys.argv[2])
split = sys.argv.index("--")
lines, cmd = sys.argv[3:split], sys.argv[split + 1:]

pid, fd = pty.fork()
if pid == 0:
    os.execvp(cmd[0], cmd)
fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))

buf = bytearray()


def read_for(secs):
    end = time.time() + secs
    while time.time() < end:
        ready, _, _ = select.select([fd], [], [], 0.2)
        if not ready:
            continue
        try:
            data = os.read(fd, 65536)
        except OSError:
            return False
        if not data:
            return False
        buf.extend(data)
        if b"\x1b[6n" in data:
            os.write(fd, b"\x1b[1;1R")
        if b"\x1b]11;?" in data:
            os.write(fd, b"\x1b]11;rgb:0000/0000/0000\x1b\\")
    return True


read_for(startup)
os.write(fd, b"".join(line.encode() + b"\r" for line in lines))
deadline = time.time() + 120
status = None
while time.time() < deadline:
    alive = read_for(1)
    wpid, st = os.waitpid(pid, os.WNOHANG)
    if wpid:
        status = os.waitstatus_to_exitcode(st)
        break
    if not alive:
        _, st = os.waitpid(pid, 0)
        status = os.waitstatus_to_exitcode(st)
        break
if status is None:
    os.kill(pid, 9)
    os.waitpid(pid, 0)
    status = 124
with open(out_path, "wb") as f:
    f.write(bytes(buf))
sys.exit(status)
