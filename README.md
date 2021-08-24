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
All components are managed by systemd.

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

## Cheat sheet
### Show stored queries via CLI
Login postgres:

```
% docker-compose -f /usr/local/etc/interceptor/docker-compose.yml exec postgres psql -d interceptor -U vsix
psql (13.4 (Debian 13.4-1.pgdg100+1))
Type "help" for help.

interceptor=# 
```

Show tables:

```
interceptor=# \dt+
                            List of relations
 Schema |    Name    | Type  | Owner | Persistence | Size  | Description 
--------+------------+-------+-------+-------------+-------+-------------
 public | query_logs | table | vsix  | permanent   | 20 MB | 
(1 row)
```

Show the number of stored queries:

```
interceptor=# SELECT COUNT(*) FROM query_logs;
 count  
--------
 190315
(1 row)
```

List all clients' addresses captured from the host:

```
interceptor=# SELECT DISTINCT(src_ip) FROM query_logs;
                src_ip                
--------------------------------------
 2001:200:e20:100:20b4:db4b:c060:dcb7
 2001:200:e00:b0::110
 2001:200:e20:20:5054:fe08:4c7e:c08f
 2001:200:e20:20:8261:5fff:fe06:76f
 2001:200:e20:100:995a:21a9:db18:db62
 2001:200:e20:100:65e4:5dd1:86a1:7639
(6 rows)
```

Show **10 most recent** queries - replace `DESC` with `ASC` to show the oldest.

```
interceptor=# SELECT received_at, src_ip, dst_ip, src_port, query_string, query_type FROM query_logs ORDER BY received_at DESC LIMIT 10;
          received_at          |                src_ip                |         dst_ip         | src_port |          query_string           | query_type 
-------------------------------+--------------------------------------+------------------------+----------+---------------------------------+------------
 2021-08-24 14:58:02.387654+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    32962 | signaler-pa.clients6.google.com | AAAA
 2021-08-24 14:58:02.385579+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    32962 | signaler-pa.clients6.google.com | A
 2021-08-24 14:58:01.27995+09  | 2001:200:e00:b0::110                 | 2001:4860:4860::6464   |    40752 | github.com                      | AAAA
 2021-08-24 14:58:01.277657+09 | 2001:200:e00:b0::110                 | 2001:4860:4860::6464   |    55013 | github.com                      | A
 2021-08-24 14:58:00.028033+09 | 2001:200:e20:100:20b4:db4b:c060:dcb7 | 2001:200:e00:b11::6464 |    64533 | ssl.gstatic.com                 | A
 2021-08-24 14:58:00.025753+09 | 2001:200:e20:100:20b4:db4b:c060:dcb7 | 2001:200:e00:b11::6464 |    57575 | ssl.gstatic.com                 | AAAA
 2021-08-24 14:57:59.835872+09 | 2001:200:e20:100:20b4:db4b:c060:dcb7 | 2001:200:e00:b11::6464 |    20262 | docs.google.com                 | AAAA
 2021-08-24 14:57:59.833792+09 | 2001:200:e20:100:20b4:db4b:c060:dcb7 | 2001:200:e00:b11::6464 |    65360 | docs.google.com                 | A
 2021-08-24 14:57:59.35588+09  | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    44963 | api.software.com                | AAAA
 2021-08-24 14:57:59.353761+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    44963 | api.software.com                | A
```

Count the number of A and AAAA requests **per FQDN** and show the **top 10** domains - replace `query_string` with `src_ip` to show per clients.

```
interceptor=# SELECT query_string, COUNT(*) AS total, COUNT(*) FILTER(WHERE query_type='A') AS a, COUNT(*) FILTER(WHERE query_type='AAAA') AS aaaa FROM query_logs GROUP BY query_string ORDER BY total DESC LIMIT 10;
                     query_string                     | total |  a  | aaaa 
------------------------------------------------------+-------+-----+------
 api.software.com                                     |  1282 | 641 |  641
 github.com                                           |   132 |  66 |   66
 signaler-pa.clients6.google.com                      |   100 |  50 |   50
 vortex.data.microsoft.com                            |    82 |  41 |   41
 ws.todoist.com                                       |    80 |  24 |   40
 ssl.gstatic.com                                      |    76 |  38 |   38
 mcs-spinnaker-2103948255.us-east-2.elb.amazonaws.com |    66 |  22 |   23
 www.google.com                                       |    59 |  23 |   34
 ipv4only.arpa                                        |    51 |   0 |   51
 play.google.com                                      |    48 |  24 |   24
(10 rows)

interceptor=# SELECT src_ip, COUNT(*) AS total, COUNT(*) FILTER(WHERE query_type='A') AS a, COUNT(*) FILTER(WHERE query_type='AAAA') AS aaaa FROM query_logs GROUP BY src_ip ORDER BY total DESC LIMIT 10;
                src_ip                | total |  a   | aaaa 
--------------------------------------+-------+------+------
 2001:200:e20:20:8261:5fff:fe06:76f   |  2858 | 1422 | 1422
 2001:200:e20:100:20b4:db4b:c060:dcb7 |  1347 |  162 |  590
 2001:200:e20:100:d59a:c6b5:dfe9:5259 |   462 |   56 |  297
 2001:200:e20:100:65e4:5dd1:86a1:7639 |   373 |   11 |  207
 2001:200:e00:b0::110                 |   112 |   56 |   56
 2001:200:e20:100:995a:21a9:db18:db62 |   100 |    3 |   52
 2001:200:e20:20:5054:fe08:4c7e:c08f  |    18 |    7 |   11
(7 rows)
```

Calculate the ratio of A's query count to AAAA's count normalized by the total, i.e., a degree of IPv4 dependency, and show the **worst 10** clients.

```
interceptor=# SELECT src_ip, (COUNT(*) FILTER(WHERE query_type='A') - COUNT(*) FILTER(WHERE query_type='AAAA')) * 100 / COUNT(*) AS v4_dependency FROM query_logs GROUP BY src_ip ORDER BY v4_dependency DESC LIMIT 10;
                src_ip                | v4_dependency 
--------------------------------------+---------------
 2001:200:e20:20:8261:5fff:fe06:76f   |             0
 2001:200:e00:b0::110                 |             0
 2001:200:e20:100:20b4:db4b:c060:dcb7 |           -28
 2001:200:e20:20:5054:fe08:4c7e:c08f  |           -33
 2001:200:e20:100:995a:21a9:db18:db62 |           -49
 2001:200:e20:100:65e4:5dd1:86a1:7639 |           -50
 2001:200:e20:100:d59a:c6b5:dfe9:5259 |           -52
```

## License
This product is licensed under [The 2-Clause BSD License](https://opensource.org/licenses/BSD-2-Clause) - see the [LICENSE](LICENSE) file for details.
