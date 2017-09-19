# Filesystem liveness exporter

This is an extremely simple filesystem liveness exporter that checks for
hung filesystems and emits the result of those checks as metrics to your
Prometheus master or other monitoring system.

Checks are performed in goroutines, a goroutine per filesystem each, to
prevent the exporter from hanging.   If your filesystem responds to
`readdir()` on its mount point when the underlying backing device is
unresponsive (for example, because the `readdir()` is cached), then
the test result will not be faithful.

This exporter exports two metrics:

* `vfs_filesystem_live`: a gauge 1 or 0 whether the file system has
  responded to `readdir()` or `read()` without hanging under the
  timeout specified on the command line (default 5 seconds).
* `vfs_filesystem_scan_duration_seconds` a gauge measuring how long
  the file system took to respond to the `readdir()` request, if
  it was responding at all.

## Help

Run `filesystem_liveness_exporter -?` to get information.

The command line parameters that can be used are:

* -check.timeout: specify how many seconds the exporter should wait
  for filesystem responses.
* -check.fstypes: a comma-separated list of allowed file system
  types to poll, defaulting to the most common networked file
  systems. If passed an empty string, all file systems will be
  allowed.
* -check.read-file: the name of a file to concatenate to the
  mount point and read to check that the file system works.
* -web.listen-address: a standard host:port or :port address to
  listen on.
* -verbose: whether to print verbose liveness checks.

## Behavioral and security notes

If you do not pass `-check.read-file`, then the program will
attempt to `readdir()` the mount point of each matching file
system type currently mounted upon each `/metrics` check.
If you do pass it, then the `read()` call will be used against
the passed file name, under each mount point.

If you pass `-check.read-file <somefilename>` and the file in
question does not exist under the checked mount points, the
metrics for that mount point will be omitted from the HTTP
output of the program.

If the checked mount point or the checked file return `EPERM`
or `ENOTDIR`, the metrics will consider the mount point alive.

The check happens in a subprocess, which gets killed if the
check extends past the `timeout` parameter.

Take real good care not to allow file system types that users
may mount or write on, especially if you use the
`-check.read-file` flag.

## License

This program is distributed under the [Apache 2.0](LICENSE) license.
