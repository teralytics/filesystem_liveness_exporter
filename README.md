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
  responded to `readdir()` without hanging under the timeout
  specified on the command line (default 5 seconds).
* `vfs_filesystem_scan_duration_seconds` a gauge measuring how long
  the file system took to respond to the `readdir()` request, if
  it was responding at all.

## Help

Run `filesystem_liveness_exporter -?` to get information.

The command line parameters that can be used are:

* -timeout: specify how many seconds the exporter should wait for
  filesystem responses.
* -fstypes: a comma-separated list of allowed file system types
  to poll, defaulting to the most common networked file systems.
* -web.listen-address: a standard host:port or :port address to
  listen on.
* -verbose: whether to print verbose liveness checks.

## License

This program is distributed under the [Apache 2.0](LICENSE) license.
