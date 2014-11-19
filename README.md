gofluent
========

This program is something acting like fluentd rewritten in Go.

Introduction
========

Fluentd is originally written in CRuby, which has too many dependecies.

I hope fluentd to be simpler and cleaner, as its main feature is simplicity and rubostness.

Installation
========

Follow steps below and have fun!

```
go get github.com/hnlq715/gofluent
go build
```

Architecture
========
```
    +---------+     +---------+     +---------+     +---------+
    | server1 |     | server2 |     | server3 |     | serverN |
    |---------|     |---------|     |---------|     |---------|
    |syslog-ng|     |syslog-ng|     |syslog-ng|     |syslog-ng|
    |---------|     |---------|     |---------|     |---------|
    |gofluent |     |gofluent |     |gofluent |     |gofluent |
    +---------+     +---------+     +---------+     +---------+
        |               |               |               |
         -----------------------------------------------
                                |
                                | HTTP POST
                                V
                        +-----------------+
                        |                 |
                        |      Httpmq     |
                        |                 | 
                        +-----------------+
                                |
                                | HTTP GET
                                V 
                        +-----------------+                 +-----------------+
                        |                 |                 |                 |
                        |   Preprocessor  | --------------> |     Storage     |
                        |                 |                 |                 | 
                        +-----------------+                 +-----------------+
```

Implementation
========
##Overview
```
Input -> Router -> Output
```
##Data flow
```
                        -------<-------- 
                        |               |
                        V               | generate pool
       InputRunner.inputRecycleChan     | recycling
            |           |               |     ^   
            |            ------->-------       \ 
            |               ^           ^       \
    InputRunner.inChan      |           |        \
            |               |           |         \
            |               |           |          \
    consume |               |           |           \
            V               |           |            \
          Input(Router.inChan) ---->  Router ----> (Router.outChan)Output.inChan

```

Plugins
========
##Tail Input Plugin
The in_tail input plugin allows gofluent to read events from the tail of text files. Its behavior is similar to the tail -F command.

####Example Configuration
in_tail is included in gofluent’s core. No additional installation process is required.
```
<source>
  type tail
  path /var/log/json.log
  pos_file /var/log/json.log.pos
  tag apache.access
  format json
</source>
```
####type (required)
The value must be tail.
####tag (required)
The tag of the event.
####path (required)
The paths to read.
####format (required)
The format of the log. It is the name of a template or regexp surrounded by ‘/’.
The regexp must have at least one named capture (?P\<NAME\>PATTERN).

The following templates are supported:
- regexp
- json
One JSON map, per line. This is the most straight forward format :).
```
format json
```
####pos_file (highly recommended)
This parameter is highly recommended. gofluent will record the position it last read into this file.
```
pos_file /var/log/access.log.pos
```

##Httpsqs Output Plugin
The out_httpsqs output plugin allows gofluent to send data to httpsqs mq.
####Example Configuration
out_httpsqs is included in gofluent’s core. No additional installation process is required.
```
<match httpsqs.**>
  type httpsqs
  host localhost
  port 1218
  flush_interval 10
</match>
```
####type (required)
The value must be httpsqs.
####host (required)
The output target host ip.
####port (required)
The output target host port.
####auth (highly recommended)
The auth password for httpsqs.
####flush_interval
The flush interval for sending data to httpsqs.
####gzip
The gzip switch, default is on.

##Forward Output Plugin
The out_forward output plugin allows gofluent to forward events to another gofluent.
####Example Configuration
out_forward is included in gofluent’s core. No additional installation process is required.
```
<match forward.**>
  type forward
  host localhost
  port 1218
  flush_interval 10
</match>
```
####type (required)
The value must be forward.
####host (required)
The output target host ip.
####port (required)
The output target host port.
####flush_interval
The flush interval for sending data to httpsqs.
####connect_timeout
The connect timeout value.

##Stdout Output Plugin
The out_stdout output plugin allows gofluent to print events to stdout.
####Example Configuration
out_stdout is included in gofluent’s core. No additional installation process is required.
```
<match stdout.**>
  type stdout
</match>
```
####type (required)
The value must be stdout.

