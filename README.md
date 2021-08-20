# dns-query-interceptor
Tiny program intercepting DNS queries

[![asciicast](https://asciinema.org/a/431381.svg)](https://asciinema.org/a/431381?autoplay=1)

## Use alone
Golang compiler and `libpcap-dev` are needed to build - you may get the latest binary from [Releases](https://github.com/wide-vsix/dns-query-interceptor/releases).

```
% git clone --depth 1 https://github.com/wide-vsix/dns-query-interceptor
% cd dns-query-interceptor
% make build
% sudo cp bin/interceptor /usr/local/bin/interceptor
```

Following options are available:

```
% interceptor -h
-i, --dev string                Capturing interface name
-q, --quiet                     Suppress standard output
    --db-host string            Postgres server address to store queries (e.g., localhost:5432)
    --db-name string            Database name to store queries
    --db-user string            Username to login DB
    --db-password-file string   Path of plaintext password file
-h, --help                      Show help message
-v, --version                   Show build version
```

## Use with PostgreSQL

**CAUTION:** Defaults of this repository are for the vSIX Access Service.

```
% echo -n 'VSIX_STANDARD_PASSWORD' | sha256sum | awk '{print $1}' > .secrets/db_password.txt
% make install
% sudo systemctl start dns-query-interceptor@vsix.service
```

Uninstall from systemd and purge the database - note that this is a destructive operation and cannot be undone.

```
% make uninstall
```

## License
This product is licensed under [The 2-Clause BSD License](https://opensource.org/licenses/BSD-2-Clause) - see the [LICENSE](LICENSE) file for details.
