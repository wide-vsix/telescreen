# telescreen
[![](https://github.com/wide-vsix/telescreen/actions/workflows/deploy_to_gpr.yml/badge.svg)](https://github.com/wide-vsix/telescreen/actions) [![](https://img.shields.io/github/issues/wide-vsix/telescreen)](https://github.com/wide-vsix/telescreen/issues) [![](https://img.shields.io/github/issues-pr/wide-vsix/telescreen)](https://github.com/wide-vsix/telescreen/pulls) [![](https://img.shields.io/github/last-commit/wide-vsix/telescreen)](https://github.com/wide-vsix/telescreen/commits) [![](https://img.shields.io/github/release/wide-vsix/telescreen)](https://github.com/wide-vsix/telescreen/releases) [![](http://img.shields.io/github/license/wide-vsix/telescreen)](LICENSE)

**Telescreen** - a tiny program intercepting DNS query-response pairs

[![asciicast](https://asciinema.org/a/435064.svg)](https://asciinema.org/a/435064?autoplay=1)

## Supported features
As of September 11, 2021, following features are available:

- Capture all DNS queries from a specified interface - you can intercept all packets to the Public DNS servers such as Google and Cloudflare
- Capture all responses to AAAA queries
- All captured packets are stored in the Postgres database

```
% telescreen -h
  -i, --dev string                Interface name
  -q, --quiet                     Suppress standard output
  -A, --with-response             Store responses to AAAA queries
  -H, --db-host string            Postgres server address to store logs (e.g., localhost:5432)
  -N, --db-name string            Database name to store
  -U, --db-user string            Username to login
  -P, --db-password-file string   Password to login - path of a plaintext password file
  -c, --container                 Run inside a container - load options from environment variables
  -h, --help                      Show help message
  -v, --version                   Show build version
```

The vSIX Access Service Team developed and maintained this software to detect IPv6 unsupported clients and servers.

## Build a telescreen binary
Golang compiler and `libpcap-dev` are needed to build - also you can get the latest binary from [Releases](https://github.com/wide-vsix/telescreen/releases).

```
% git clone --depth 1 https://github.com/wide-vsix/telescreen
% cd telescreen
% make build install
```

### Use static linking
You can build a statically linked binary using docker. In this way, all you need to prepare is a docker environment. If you want to do the same thing in your native environment, you need to compile `libpcap.a` beforehand and write the library path to Makefile. Detailed procedure is described in [Dockerfile](Dockerfile).

```
% make build-static-docker install
% docker images | grep wide-vsix/telescreen
% docker run --rm --network host wide-vsix/telescreen:21.09.11-952e89d -i vsix -A
```

You can also pull prebuilt docker image from [GitHub Package Registry](https://github.com/orgs/wide-vsix/packages?repo_name=telescreen).

## Construct a telescreen network
The telescreens installed on multiple router VMs with a shared remote database

**CAUTION:** Defaults of this repository are for the vSIX Access Service and cannot be reused in another environment.

### Setup database
Just run the following on the database host:

```
% echo -n 'VSIX_STANDARD_PASSWORD' | sha256sum | awk '{print $1}' > .secrets/db_password.txt
% make install-database
```

Wait a few tens of seconds until the Postgres container is fully up and running, then go to the agent setup.

### Setup agent
Build a telescreen binary with static linking somewhere and distribute it to all the hosts capturing DNS packets. Create a secret to login database and install telescreen service.

```
% echo -n 'VSIX_STANDARD_PASSWORD' | sha256sum | awk '{print $1}' > .secrets/db_password.txt
% make install-agent
% sudo systemctl start telescreen@vsix.service
```

**NOTE:** On VyOS, `systemctl enable` seems to fail, but it actually works. Remember to run `systemctl restart` after every system reboot.

### Uninstall
Run the following on every agent and database host to uninstall telescreen from systemd and purge the stored packets - note that this is a destructive operation and cannot be undone.

```
% make uninstall
```

## Cheat sheet
### Show stored queries and responses via CLI
Login postgres:

```
% docker-compose -f /var/lib/telescreen/docker-compose.yml exec postgres psql -d telescreen -U vsix
psql (13.4 (Debian 13.4-1.pgdg100+1))
Type "help" for help.

telescreen=# 
```

Show tables:

```
telescreen=# \dt+
                              List of relations
 Schema |     Name      | Type  | Owner | Persistence |  Size  | Description 
--------+---------------+-------+-------+-------------+--------+-------------
 public | query_logs    | table | vsix  | permanent   | 176 MB | 
 public | response_logs | table | vsix  | permanent   | 70 MB  | 
(2 rows)
```

Show the number of stored queries:

```
telescreen=# SELECT COUNT(*) FROM query_logs;
  count  
---------
 1581177
(1 row)
```

List all clients' addresses captured from the host:

```
telescreen=# SELECT COUNT(DISTINCT(src_ip)) FROM query_logs;
 count 
-------
  7333
(1 row)
```

```
telescreen=# SELECT DISTINCT(src_ip) FROM query_logs LIMIT 10;
                src_ip                
--------------------------------------
 2001:200:e20:110:d5e9:19a9:2024:e2b0
 2001:200:e20:160:688b:f550:d98d:6d06
 2001:200:e20:160:f4ab:3784:5ce2:de72
 2001:200:e20:110:e54f:6f32:9e16:4f6c
 2001:200:e20:160:e16a:bad9:2bfb:1bc4
 2001:200:e20:110:6096:dc2:978d:560a
 2001:200:e20:160:ed87:f8d1:45fe:6e7c
 2001:200:e20:110:2cb9:ea30:7f63:fc97
 2001:200:e20:110:a8be:1599:4820:ca4b
 2001:200:e20:160:70e3:4b33:aa50:444
(10 rows)
```

Show **10 most recent** queries - replace `DESC` with `ASC` to show the oldest.

```
telescreen=# SELECT received_at, src_ip, dst_ip, src_port, query_string, query_type FROM query_logs ORDER BY received_at DESC LIMIT 10;
          received_at          |                src_ip                |         dst_ip         | src_port |              query_string               | query_type 
-------------------------------+--------------------------------------+------------------------+----------+-----------------------------------------+------------
 2021-09-11 15:41:45.768538+09 | 2001:200:e20:1080:98ec:af55:c03a:94f | 2001:200:e00:b11::6464 |    55303 | db._dns-sd._udp.proelbtn.com            | PTR
 2021-09-11 15:41:45.764818+09 | 2001:200:e20:1080:98ec:af55:c03a:94f | 2001:200:e00:b11::6464 |    57668 | b._dns-sd._udp.proelbtn.com             | PTR
 2021-09-11 15:41:45.761569+09 | 2001:200:e20:1080:98ec:af55:c03a:94f | 2001:200:e00:b11::6464 |    63953 | db._dns-sd._udp.0.0.16.172.in-addr.arpa | PTR
 2021-09-11 15:41:45.758455+09 | 2001:200:e20:1080:98ec:af55:c03a:94f | 2001:200:e00:b11::6464 |    54990 | b._dns-sd._udp.0.0.16.172.in-addr.arpa  | PTR
 2021-09-11 15:41:42.300656+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    43444 | safebrowsing.googleapis.com             | AAAA
 2021-09-11 15:41:42.298583+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    43444 | safebrowsing.googleapis.com             | A
 2021-09-11 15:41:41.592282+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    47468 | vortex.data.microsoft.com               | AAAA
 2021-09-11 15:41:41.586375+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    47468 | vortex.data.microsoft.com               | A
 2021-09-11 15:41:40.284799+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    38866 | mailv3.m.titech.ac.jp                   | AAAA
 2021-09-11 15:41:40.282509+09 | 2001:200:e20:20:8261:5fff:fe06:76f   | 2001:4860:4860::6464   |    38866 | mailv3.m.titech.ac.jp                   | A
(10 rows)
```

Show 10 most recent queries **from specified prefixes** - replace `DESC` with `ASC` to show the oldest. See [network operators](https://www.postgresql.org/docs/current/functions-net.html) for details.

```
telescreen=# SELECT received_at, src_ip, dst_ip, src_port, query_string, query_type FROM query_logs WHERE src_ip << '2001:200:e00:d10::/56' ORDER BY received_at DESC LIMIT 10;
          received_at          |                src_ip                |         dst_ip         | src_port |                              query_string                              | query_type 
-------------------------------+--------------------------------------+------------------------+----------+------------------------------------------------------------------------+------------
 2021-09-09 07:27:15.176486+09 | 2001:200:e00:d10:dcf7:6f5b:f7a7:694  | 2001:200:e00:b11::6464 |    64493 | slack.com                                                              | AAAA
 2021-09-09 07:27:15.170375+09 | 2001:200:e00:d10:dcf7:6f5b:f7a7:694  | 2001:200:e00:b11::6464 |    60935 | slack.com                                                              | Unknown
 2021-09-09 07:27:15.161474+09 | 2001:200:e00:d10:dcf7:6f5b:f7a7:694  | 2001:200:e00:b11::6464 |    51696 | prod-envoy-wss-nlb-3-e1fc9c96272f3076.elb.ap-northeast-1.amazonaws.com | Unknown
 2021-09-09 07:27:15.157368+09 | 2001:200:e00:d10:dcf7:6f5b:f7a7:694  | 2001:200:e00:b11::6464 |    49734 | wss-mobile.slack.com                                                   | AAAA
 2021-09-09 07:27:15.154597+09 | 2001:200:e00:d10:dcf7:6f5b:f7a7:694  | 2001:200:e00:b11::6464 |    63520 | wss-mobile.slack.com                                                   | Unknown
 2021-09-09 07:27:13.79021+09  | 2001:200:e00:d10:ff1a:a3b8:1c98:bb2d | 2001:4860:4860::6464   |    56586 | connectivity-check.ubuntu.com                                          | AAAA
 2021-09-09 07:27:12.045588+09 | 2001:200:e00:d10:716d:b40e:eb24:c18c | 2001:200:e00:b11::6464 |    60907 | ocsp2.g.aaplimg.com                                                    | AAAA
 2021-09-09 07:27:12.042266+09 | 2001:200:e00:d10:716d:b40e:eb24:c18c | 2001:200:e00:b11::6464 |    56188 | ocsp2.g.aaplimg.com                                                    | Unknown
 2021-09-09 07:27:12.013221+09 | 2001:200:e00:d10:716d:b40e:eb24:c18c | 2001:200:e00:b11::6464 |    58597 | ocsp2.apple.com                                                        | AAAA
 2021-09-09 07:27:12.010212+09 | 2001:200:e00:d10:716d:b40e:eb24:c18c | 2001:200:e00:b11::6464 |    54359 | ocsp2.apple.com                                                        | Unknown
(10 rows)
```

Count the number of A and AAAA requests **per FQDN** and show the **top 10** domains - replace `query_string` with `src_ip` to show per clients.

```
telescreen=# SELECT query_string, COUNT(*) AS total, COUNT(*) FILTER(WHERE query_type='A') AS a, COUNT(*) FILTER(WHERE query_type='AAAA') AS aaaa FROM query_logs GROUP BY query_string ORDER BY total DESC LIMIT 10;
          query_string           | total |   a   | aaaa  
---------------------------------+-------+-------+-------
 safebrowsing.googleapis.com     | 77660 | 38496 | 39102
 api.software.com                | 54867 | 27434 | 27433
 keepalive.softether.org         | 43361 | 43361 |     0
 github.com                      | 36600 | 18156 | 18367
 m.root-servers.net              | 26029 | 13299 | 12730
 slack.com                       | 22964 |  9054 | 12833
 ipv4only.arpa                   | 21570 |   559 | 21011
 signaler-pa.clients6.google.com | 19264 |  9134 | 10130
 www.apple.com                   | 17280 |  1069 | 16160
 ws.todoist.com                  | 16958 |  6377 |  8053
(10 rows)
```

```
telescreen=# SELECT src_ip, COUNT(*) AS total, COUNT(*) FILTER(WHERE query_type='A') AS a, COUNT(*) FILTER(WHERE query_type='AAAA') AS aaaa FROM query_logs GROUP BY src_ip ORDER BY total DESC LIMIT 10;
                src_ip                 | total  |   a    |  aaaa  
---------------------------------------+--------+--------+--------
 2001:200:e20:20:8261:5fff:fe06:76f    | 346252 | 175143 | 170892
 2001:200:e00:d10:b97c:bfa6:6d58:cc78  |  46079 |  21907 |  24162
 2001:200:e00:d10:b13d:f48c:5bf2:4d3d  |  35479 |   2749 |  18843
 2001:200:e00:d10:ec68:fd4c:41b:738a   |  31371 |   2367 |  20303
 2001:200:e00:d10:dcf7:6f5b:f7a7:694   |  29716 |    375 |  16839
 2001:200:e20:1070:2107:932e:9109:d15e |  28367 |  18538 |   9822
 2001:200:e00:d10:e8c5:9354:318f:197f  |  27848 |  11788 |  12787
 2001:200:e20:1080:848e:a603:b4a1:5478 |  26099 |  10819 |  11130
 2001:200:e00:b0::110                  |  24805 |  12280 |  12231
 2001:200:e20:1070:cde6:d9a1:8c70:bce3 |  24716 |  14378 |  10337
(10 rows)
```

Calculate the ratio of A's query count to AAAA's count normalized by the total, i.e., a degree of IPv4 dependency, and show the **best 10** clients - replace `ASC` with `DESC` to show the worst.

```
telescreen=# SELECT src_ip, (COUNT(*) FILTER(WHERE query_type='A') - COUNT(*) FILTER(WHERE query_type='AAAA')) * 100 / COUNT(*) AS v4_dependency FROM query_logs GROUP BY src_ip ORDER BY v4_dependency ASC LIMIT 10;
               src_ip                | v4_dependency 
-------------------------------------+---------------
 2001:200:e20:110:61:3a50:86a5:d352  |          -100
 2001:200:e20:110:47:8e1e:40b5:10b1  |          -100
 2001:200:e20:c0:f12a:42b9:8d0:f228  |          -100
 2001:200:e20:c0:b0cc:35bb:6844:65f  |          -100
 2001:200:e20:30:c47:36e5:7b6c:49b3  |          -100
 2001:200:e20:110:3c:5163:626:2e85   |          -100
 2001:200:e20:30:143c:8a38:4ef8:b245 |          -100
 64:ff9b:beaf::59f8:a5a4             |          -100
 2001:200:e20:30:d458:a9ba:dd24:1452 |          -100
 2001:200:e20:110:ac:bd11:a74d:1899  |          -100
(10 rows)
```

Sort domains supporting IPv6 by their popularity - add `NOT` to show IPv4 only domains.

```
telescreen=# SELECT query_string, COUNT(*) AS total FROM response_logs WHERE ipv6_ready GROUP BY query_string ORDER BY total DESC LIMIT 10;
           query_string            | total 
-----------------------------------+-------
 safebrowsing.googleapis.com       | 39103
 m.root-servers.net                | 11533
 signaler-pa.clients6.google.com   | 10132
 play.google.com                   |  8455
 ssl.gstatic.com                   |  8103
 dns64.dns.google                  |  7691
 e6858.dscx.akamaiedge.net         |  6947
 zabbix01.fujisawa.vsix.wide.ad.jp |  6367
 www.google.com                    |  5891
 gateway.fe.apple-dns.net          |  5774
(10 rows)
```

```
telescreen=#  SELECT query_string, COUNT(*) AS total FROM response_logs WHERE NOT ipv6_ready GROUP BY query_string ORDER BY total DESC LIMIT 10;
                     query_string                     | total 
------------------------------------------------------+-------
 github.com                                           | 34375
 api.software.com                                     | 27431
 ipv4only.arpa                                        | 20872
 apple.com                                            | 12902
 slack.com                                            | 12805
 edgeapi.slack.com                                    |  4894
 e4478.a.akamaiedge.net                               |  4592
 ss-prod-an1-notif-8.aws.adobess.com                  |  4590
 mcs-spinnaker-2103948255.us-east-2.elb.amazonaws.com |  4249
 d27xxe7juh1us6.cloudfront.net                        |  4216
(10 rows)
```

## Maintainers
This repository is maintained by the vSIX Access Service Team. Followings are responsible for reviewing pull requests:

- **miya** - *Author of the initial release* [@mi2428](https://github.com/mi2428)

See also the list of [contributors](https://github.com/wide-vsix/vsixpi/graphs/contributors) who participated in this project

## License
This product is licensed under [The 2-Clause BSD License](https://opensource.org/licenses/BSD-2-Clause) - see the [LICENSE](LICENSE) file for details.
