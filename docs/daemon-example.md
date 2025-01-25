# NDNd Daemon Usage Example

These examples demonstrate how to use NDNd to create a small NDN network.

NDNd is the combined daemon with YaNFD (an NDN Forwarder) and ndn-dv (an NDN Router). While the example below uses the combined daemon, you can also use YaNFD and ndn-dv separately (or, for example, [NFD](https://github.com/named-data/NFD) with ndn-dv). The configuration in this case is identical, with the exception of running the two instances separately.

As a prerequisite, you must have NDNd installed on all nodes in the network. See the project [README](https://github.com/named-data/ndnd) for instructions on how to use the prebuilt binaries or build from source.

## File Transfer Example

This example demonstrates a 2-node network where one node (`bob`) fetches a file from the other node (`alice`).

On both nodes, we use the simple configuration file below. Put this config in a `conf.yml` file on each node. Remember to replace `<node-name>` with the actual node name (in this case, `alice` and `bob`).

Full configuration examples with documentation for routing (`dv`) and forwarding (`fw`) can be found [here](../dv/dv.sample.yml) and [here](../fw/fw.sample.yml) respectively.

```yaml
dv:
  network: /testnet
  router: /testnet/<node-name>
  keychain: "insecure"

fw:
  faces:
    udp:
      enabled_unicast: true
      enabled_multicast: false
      port_unicast: 6363
    tcp:
      enabled: true
      port_unicast: 6363
    websocket:
      enabled: false
  fw:
    threads: 2
```

1. The `network` name must be the same on all nodes in the network.
2. The `router` name must be unique for each node in the network.
3. The `faces` section configures the faces that the forwarder will listen on. In this case, we enable UDP and TCP faces on port 6363. The unix face is enabled by default and listens on `/run/nfd/nfd/sock`.
4. For this simple example, we disable routing security with `insecure`, which is not recommended for production use.

Once the configuration file is in place, start the NDNd daemon on each node:

```sh
# root permissions are requird to bind to the default unix socket
sudo ndnd daemon conf.yml
```

Once the daemons are running, we create a routing neighbor relationship between the two nodes. On `alice`, run:

```sh
# if udp is blocked, use tcp instead
ndnd dv link create "udp://<bob-ip>:6363"
```

After a few seconds, logs should show up on both nodes indicating that the neighbor relationship has been established.

Now, we can start serving the file on `alice` using the `put` tool:

```sh
# create a file to serve
echo "Hello, NDN!" > /tmp/hello.txt

# serve the file using put
# -expose enables advertising the file to the routing protocol
ndnd put -expose /alice/hello < /tmp/hello.txt
```

Logs on both sides should show the newly announced prefix `/alice/hello`.
On `bob`, we can now fetch the file using the `cat` tool:

```sh
# fetch the file using cat
ndnd cat /alice/hello > /tmp/hello.txt

# check the contents of the file
cat /tmp/hello.txt
```

This concludes the file transfer example. You can now experiment with more complex topologies by adding more nodes and creating more routing relationships. The routing protocol will automatically propagate prefixes to all nodes in the network.
