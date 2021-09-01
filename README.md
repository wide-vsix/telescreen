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

On database host:

```
% echo -n 'VSIX_STANDARD_PASSWORD' | sha256sum | awk '{print $1}' > .secrets/db_password.txt
% make install-db
```

On monitoring hosts:

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
 Schema |     Name      | Type  | Owner | Persistence | Size  | Description 
--------+---------------+-------+-------+-------------+-------+-------------
 public | query_logs    | table | vsix  | permanent   | 67 MB | 
 public | response_logs | table | vsix  | permanent   | 30 MB | 
(2 rows)
```

Show the number of stored queries:

```
interceptor=# SELECT COUNT(*) FROM query_logs;
 count  
--------
 611757
(1 row)
```

List all clients' addresses captured from the host:

```
interceptor=# SELECT COUNT(DISTINCT(src_ip)) FROM query_logs;
 count 
-------
  3084
(1 row)
```

```
interceptor=# SELECT DISTINCT(src_ip) FROM query_logs LIMIT 10;
                src_ip                
--------------------------------------
 2001:200:e20:110:d501:9e30:96c3:dbee
 2001:200:e20:110:54d5:c0ea:1e46:b18
 2001:200:e20:30:b09c:75d6:b8a:dc61
 2001:200:e20:30:4a2:b1f5:60ef:66a8
 2001:200:e20:110:7dd9:2164:c54b:fdcc
 2001:200:e20:110:1574:f317:f468:c814
 2001:200:e20:110:18b5:9b27:f1a2:767
 2001:200:e20:110:a158:98f6:2bc8:db99
 2001:200:e20:110:8870:ecd3:59a2:336a
 2001:200:e20:30:9458:4542:f6a7:6921
(10 rows)
```

Show **10 most recent** queries - replace `DESC` with `ASC` to show the oldest.

```
interceptor=# SELECT received_at, src_ip, dst_ip, src_port, query_string, query_type FROM query_logs ORDER BY received_at DESC LIMIT 10;
          received_at          |               src_ip               |        dst_ip        | src_port |          query_string           | query_type 
-------------------------------+------------------------------------+----------------------+----------+---------------------------------+------------
 2021-09-01 17:35:04.915184+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    41867 | www.amazon.co.jp                | AAAA
 2021-09-01 17:35:04.912689+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    41867 | www.amazon.co.jp                | A
 2021-09-01 17:35:03.286391+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    57975 | vortex.data.microsoft.com       | AAAA
 2021-09-01 17:35:03.284447+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    57975 | vortex.data.microsoft.com       | A
 2021-09-01 17:35:01.850723+09 | 2001:200:e00:b0::110               | 2001:4860:4860::6464 |    43954 | github.com                      | AAAA
 2021-09-01 17:35:01.848449+09 | 2001:200:e00:b0::110               | 2001:4860:4860::6464 |    50450 | github.com                      | A
 2021-09-01 17:34:57.710979+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    35531 | signaler-pa.clients6.google.com | AAAA
 2021-09-01 17:34:57.708422+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    35531 | signaler-pa.clients6.google.com | A
 2021-09-01 17:34:48.079762+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    40454 | vortex.data.microsoft.com       | AAAA
 2021-09-01 17:34:48.076557+09 | 2001:200:e20:20:8261:5fff:fe06:76f | 2001:4860:4860::6464 |    40454 | vortex.data.microsoft.com       | A
(10 rows)
```

Count the number of A and AAAA requests **per FQDN** and show the **top 10** domains - replace `query_string` with `src_ip` to show per clients.

```
interceptor=# SELECT query_string, COUNT(*) AS total, COUNT(*) FILTER(WHERE query_type='A') AS a, COUNT(*) FILTER(WHERE query_type='AAAA') AS aaaa FROM query_logs GROUP BY query_string ORDER BY total DESC LIMIT 10;
           query_string            | total  |   a   | aaaa  
-----------------------------------+--------+-------+-------
 api.software.com                  | 130899 | 65450 | 65449
 safebrowsing.googleapis.com       |  31048 | 15524 | 15524
 github.com                        |  18273 |  9184 |  9074
 signaler-pa.clients6.google.com   |  16589 |  8294 |  8295
 ws.todoist.com                    |  11517 |  5374 |  5541
 play.google.com                   |   9415 |  4708 |  4706
 vortex.data.microsoft.com         |   8761 |  4378 |  4383
 calendar.google.com               |   8368 |  4180 |  4184
 zabbix01.fujisawa.vsix.wide.ad.jp |   8064 |  4004 |  4034
 ipv4only.arpa                     |   7745 |   134 |  7611
(10 rows)

interceptor=# SELECT src_ip, COUNT(*) AS total, COUNT(*) FILTER(WHERE query_type='A') AS a, COUNT(*) FILTER(WHERE query_type='AAAA') AS aaaa FROM query_logs GROUP BY src_ip ORDER BY total DESC LIMIT 10;
                src_ip                | total  |   a    |  aaaa  
--------------------------------------+--------+--------+--------
 2001:200:e20:20:8261:5fff:fe06:76f   | 334656 | 167789 | 166715
 2001:200:e00:b0::110                 |  17787 |   8738 |   8599
 2001:200:e20:110:9907:8096:607e:7eaa |  11122 |    447 |   8106
 2001:200:e20:110:d028:73a7:e31d:f57f |   8232 |   2739 |   2857
 2001:200:e20:110:e802:5495:bd0f:834  |   6429 |    881 |   3070
 2001:200:e20:c0:a81c:ee1f:a0b6:880c  |   5146 |   2013 |   2136
 2001:200:e20:110:a079:7d82:8ce0:b87c |   5136 |    195 |   2842
 2001:200:e20:110:6409:9133:5768:4841 |   4625 |   1282 |   1368
 2001:200:e20:30:fc4d:9ffd:54f6:f6b   |   4366 |   1666 |   2333
 2001:200:e20:110:5dfb:c750:bcd0:17e5 |   4123 |    164 |   2385
(10 rows)
```

Calculate the ratio of A's query count to AAAA's count normalized by the total, i.e., a degree of IPv4 dependency, and show the **worst 10** clients.

```
interceptor=# SELECT src_ip, (COUNT(*) FILTER(WHERE query_type='A') - COUNT(*) FILTER(WHERE query_type='AAAA')) * 100 / COUNT(*) AS v4_dependency FROM query_logs GROUP BY src_ip ORDER BY v4_dependency DESC LIMIT 10;
                 src_ip                 | v4_dependency 
----------------------------------------+---------------
 2001:200:e20:20:9744:64b2:5672:dd34    |           100
 2001:200:e20:20:e890:8fdb:7bf3:40cf    |           100
 2405:6581:800:6c10:714e:bc06:75e7:5b8b |           100
 2001:200:e20:20:639a:29f5:3a46:b464    |           100
 2001:200:e20:20:62f4:240b:a3c4:c052    |           100
 2001:200:e20:110:458:efb6:9e3c:325b    |           100
 2001:200:e20:20:6788:4a0f:9966:e82a    |           100
 2001:200:e20:20:421:c6ca:b485:12d4     |           100
 2001:200:e20:20:1b9b:a4af:ad73:969f    |           100
 2001:200:e20:20:4e1d:cd6d:6f59:c89     |            80
(10 rows)
```

Sort domains supporting IPv6 by their popularity - remove `NOT` to show IPv4 only domains.

```
interceptor=# SELECT query_string, COUNT(*) AS total FROM response_logs WHERE NOT ipv6_ready IS NULL GROUP BY query_string ORDER BY total DESC LIMIT 10;
            query_string            | total 
------------------------------------+-------
 safebrowsing.googleapis.com        | 15550
 signaler-pa.clients6.google.com    |  8296
 play.google.com                    |  4706
 calendar.google.com                |  4185
 zabbix01.fujisawa.vsix.wide.ad.jp  |  4035
 www.google.com                     |  3485
 e6858.dscx.akamaiedge.net          |  2920
 ssl.gstatic.com                    |  2920
 notify.bugsnag.com                 |  1451
 mgmt.pe01.fujisawa.vsix.wide.ad.jp |  1217
(10 rows)

interceptor=# SELECT query_string, COUNT(*) AS total FROM response_logs WHERE ipv6_ready IS NULL GROUP BY query_string ORDER BY total DESC LIMIT 10;
         query_string          | total 
-------------------------------+-------
 api.software.com              | 65448
 github.com                    | 17364
 ipv4only.arpa                 |  7988
 apple.com                     |  5003
 slack.com                     |  3873
 paper.dropbox.com             |  2846
 d27xxe7juh1us6.cloudfront.net |  2447
 edgeapi.slack.com             |  2247
 e4478.a.akamaiedge.net        |  1974
 s3.amazonaws.com              |  1816
(10 rows)
```

## License
This product is licensed under [The 2-Clause BSD License](https://opensource.org/licenses/BSD-2-Clause) - see the [LICENSE](LICENSE) file for details.
