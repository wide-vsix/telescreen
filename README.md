# dns-query-interceptor
Tiny program intercepting DNS queries

[![asciicast](https://asciinema.org/a/432218.svg)](https://asciinema.org/a/432218?autoplay=1)

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
  -i, --dev string                Interface name
  -q, --quiet                     Suppress standard output
  -A, --with-response             Store responses to AAAA queries
      --db-host string            Postgres server address to store logs (e.g., localhost:5432)
      --db-name string            Database name to store
      --db-user string            Username to login
      --db-password-file string   Password to login - path of a text file containing plaintext password
  -h, --help                      Show help message
  -v, --version                   Show build version
```

### Use a statically-linked binary
Build and run binary inside docker:

```
% make docker-build
% docker images | grep wide-vsix/dns-query-interceptor
% docker run --rm --network host wide-vsix/dns-query-interceptor:21.08.27-0c418c3 -i vsix -A
```

To build binary on your native Linux, you need to compile `libpcap.a` beforehand and write the library path to Makefile. Detailed procedure is described in Dockerfile.

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
### Show stored queries and responses via CLI
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
 Schema |     Name      | Type  | Owner | Persistence |  Size  | Description 
--------+---------------+-------+-------+-------------+--------+-------------
 public | query_logs    | table | vsix  | permanent   | 544 kB | 
 public | response_logs | table | vsix  | permanent   | 216 kB | 
(2 rows)
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
           received_at          |               src_ip               |        dst_ip        | src_port |       query_string        | query_type 
-------------------------------+------------------------------------+----------------------+----------+---------------------------+------------
 2021-08-25 17:24:21.608003+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    41148 | vortex.data.microsoft.com | AAAA
 2021-08-25 17:24:21.605671+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    41148 | vortex.data.microsoft.com | A
 2021-08-25 17:24:21.160051+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    34097 | api.software.com          | AAAA
 2021-08-25 17:24:21.157676+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    34097 | api.software.com          | A
 2021-08-25 17:24:18.084043+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    42993 | ws.todoist.com            | AAAA
 2021-08-25 17:24:18.081667+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    42993 | ws.todoist.com            | A
 2021-08-25 17:24:11.075838+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    35101 | api.software.com          | AAAA
 2021-08-25 17:24:11.073594+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    35101 | api.software.com          | A
 2021-08-25 17:24:08.960203+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    41947 | www.google.com            | AAAA
 2021-08-25 17:24:08.957728+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    41947 | www.google.com            | A
(10 rows)
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
 2001:200:e20:110:2567:dc41:6982:c766 |  2526 |  340 | 1093
 2001:200:e20:20:8261:5fff:fe06:76f   |  2475 | 1236 | 1239
 2001:200:e20:110:8585:cb6d:cfd0:9a47 |  1415 |   30 |  741
 2001:200:e00:b0::110                 |   126 |   66 |   60
 2001:200:e20:20:c16e:2e1a:89d6:5c07  |    20 |   11 |    9
 2001:200:e20:c0:8385:4b2a:d7e5:4490  |    14 |    8 |    6
 2001:200:e20:30:20c:29ff:fe51:3a     |     7 |    0 |    3
(7 rows)
```

Calculate the ratio of A's query count to AAAA's count normalized by the total, i.e., a degree of IPv4 dependency, and show the **worst 10** clients.

```
interceptor=# SELECT src_ip, (COUNT(*) FILTER(WHERE query_type='A') - COUNT(*) FILTER(WHERE query_type='AAAA')) * 100 / COUNT(*) AS v4_dependency FROM query_logs GROUP BY src_ip ORDER BY v4_dependency DESC LIMIT 10;
                 src_ip                | v4_dependency 
--------------------------------------+---------------
 2001:200:e20:c0:8385:4b2a:d7e5:4490  |            14
 2001:200:e20:20:c16e:2e1a:89d6:5c07  |            10
 2001:200:e00:b0::110                 |             5
 2001:200:e20:20:8261:5fff:fe06:76f   |             0
 2001:200:e20:110:2567:dc41:6982:c766 |           -29
 2001:200:e20:30:20c:29ff:fe51:3a     |           -42
 2001:200:e20:110:8585:cb6d:cfd0:9a47 |           -50
(7 rows)
```

Sort domains supporting IPv6 by their popularity - remove `NOT` to show IPv4 only domains.

```
interceptor=# SELECT query_string, COUNT(*) AS total FROM response_logs WHERE NOT ipv6_ready IS NULL GROUP BY query_string ORDER BY total DESC LIMIT 10;
             query_string             | total 
--------------------------------------+-------
 signaler-pa.clients6.google.com      |    61
 ssl.gstatic.com                      |    53
 todoist.com                          |    39
 play.google.com                      |    38
 www.google.com                       |    26
 calendar.google.com                  |    23
 notify.bugsnag.com                   |    22
 monkeybreadsoftware.de               |    22
 e673.dsce9.akamaiedge.net            |    18
 googlehosted.l.googleusercontent.com |    17
(10 rows)

interceptor=# SELECT query_string, COUNT(*) AS total FROM response_logs WHERE ipv6_ready IS NULL GROUP BY query_string ORDER BY total DESC LIMIT 10;
                     query_string                     | total 
------------------------------------------------------+-------
 api.software.com                                     |   456
 github.com                                           |   133
 ipv4only.arpa                                        |    54
 s3.amazonaws.com                                     |    49
 mcs-spinnaker-2103948255.us-east-2.elb.amazonaws.com |    44
 d27xxe7juh1us6.cloudfront.net                        |    30
 edgeapi.slack.com                                    |    29
 slack.com                                            |    23
 stream.pushbullet.com                                |    20
 e6987.a.akamaiedge.net                               |    20
(10 rows)
```

## License
This product is licensed under [The 2-Clause BSD License](https://opensource.org/licenses/BSD-2-Clause) - see the [LICENSE](LICENSE) file for details.
