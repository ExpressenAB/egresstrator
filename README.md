# egresstrator

### Status
[![Build Status](https://travis-ci.org/ExpressenAB/egresstrator.svg?branch=master)](https://travis-ci.org/ExpressenAB/egresstrator)

### What's this egresstrator thing?
Egresstrator will handle and set egress iptbles rules for docker containers inside their network namespaces.
Egresstrators works by monitoring containers startup events and listening for containers that starts with a magic environmnet variable set: `EGRESSTRATOR_ENABLE=1`.

If this environmnet variable is set, egresstrator will start a privileged container and set iptable rules in the container's namespace. By default the provided rule-set will set egress rules only, thus the name!

If egresstrator should be enabled, any keys defined in the environment variable `EGRESSTRATOR_ACL=key1,key2` will be set in the namespace.

Rules are defined in Consul's kv.

#### Rules
Rules will by default be picked up by consul-template from Consul's k/v at: egress/acl/*.

To create rules, make shure the `$CONSUL_HTTP_TOKEN` has read rights to the path. Then create named keys with some of the following content:

1) pure address rules in the form of `<ip>/<mask>:<proto>/<portdef>`. For example: 

```
10.50.128.0/24:tcp/20000-32000
10.50.128.0/24:tcp/80 
0.0.0.0/0:tcp/8080
10.50.128.0/24:tcp/1337-1600
```

It is also possible to refer to services in Consul's service catalog: `<tag>.<service>` For example:
```
production.postgresql
```

Any keys found in egress/acl/* that matches `EGRESSTRATOR_ACL=key1,key2` will be applied as iptable rules.

### Running egresstrator
Egresstrator is best run through systemd, and takes the below options:

```shell
NAME:
   egresstrator - Set egress rules in network namespaces.
   Enable egresstrator with EGRESSTRATOR_ENABLE=1 in your container.
   Specify egress rules with EGRESSTRATOR_ACL=myservice,otherservice

USAGE:
   egresstrator [global options] command [command options] [arguments...]

VERSION:
   0.2.0

COMMANDS:
     set      Set egress rules on specified container
     clear    Clear egress rules on specified container
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --consul value, -c value        Consul address (default: "127.0.0.1:8500") [$CONSUL_HTTP_ADDR]
   --consul-token value, -t value  Consul token [$CONSUL_HTTP_TOKEN]
   --kv-path value, -k value       Consul K/V path for egress ACL's (default: "egress/acl/") [$CONSUL_PATH]
   --template value, -f value      Custom consul template [$CONSUL_TEMPLATE]
   --image value, -i value         Docker image name
   --ssl                           Use SSL when accessing Consul
   --ssl-ca-cert value             Path to a custom SSL CA cert to use when accessing Consul
   --help, -h                      show help
   --version, -v                   print the version
```


