# dyn.direct

Free automated subdomains for dynamic DNS.

dyn.direct offers a free and open source API to automatically acquire subdomains for any use case, with HTTPS support.

### CLI Tool

You can install the cli tool via:

```
go install github.com/csnewman/dyndirect/cli/cmd/dyndirect@v0.1.0
```

A reverse proxy can be started by:

```
$ dyndirect proxy 8123 http://example.com --override-host
                                                                                                              
  Active: https://127-0-0-1-v4.ce555284-2c10-4786-ace6-05e2dab77bd8.v1.dyn.direct:8123 -> http://example.com  
  You can also visit http://127.0.0.1:8123 to be redirected to HTTPS.                                         
                                                                                                              
  ............................................................                                                
  ............................................................                                                
  ............................................................                                                
  ............................................................                                                
  ............................................................                                                
  [2023-07-16 22:56:46] 127.0.0.1:47268: /                                                                    
  [2023-07-16 22:56:46] 127.0.0.1:47268: /favicon.ico                                                         
  [2023-07-16 22:56:48] 127.0.0.1:47268: /                                                                    
  [2023-07-16 22:56:48] 127.0.0.1:59010: /                                                                    
  [2023-07-16 22:56:48] 127.0.0.1:59016: /                                                                    
                                                                                                              
  Press q to exit
```

The proxy will accept both HTTP and HTTPS connections. Any connections made via HTTP will be redirected to
the `dyn.direct` allocated subdomain automatically.

The tool will bind to all IPs on the host.

### API & Library

Details about the api can be found on [https://dyn.direct/](https://dyn.direct/).
