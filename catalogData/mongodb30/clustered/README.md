# Clustered MongoDb

## Preparation

This service is served as unconfigured cluster (each node doesn't know about each other).
To configure it we need to get it's credentials. You can achieve that by binding it to some app.

This is example of environments returned by binded app:
```json
{
  "VCAP_SERVICES": {
    "mongodb30-multinode": [
      {
        "credentials": {
          "dbname": "admin",
          "password": "PnwUYmYPt9",
          "replicaSetName": "replica",
          "replicas": [
            {
              "host": "x5c9a8487b52d4-node0.service.consul",
              "ports": {
                "27017/tcp": "30274"
              },
              "replicaName": "x5c9a8487b52d4-node0"
            },
            {
              "host": "x5c9a8487b52d4-node1.service.consul",
              "ports": {
                "27017/tcp": "30263"
              },
             "replicaName": "x5c9a8487b52d4-node1"
            },
            {
              "host": "x5c9a8487b52d4-node2.service.consul",
              "ports": {
                "27017/tcp": "32045"
              },
              "replicaName": "x5c9a8487b52d4-node2"
            }
          ],
          "serviceName": "mongodb-clustered",
          "username": "sUkwJiZoEh"
        },
        "label": "mongodb30-multinode",
        "name": "test-service-mongo",
        "plan": "simple",
        "tags": [
          "mongo",
          "k8s",
          "clustered"
        ]
      }
    ]
  }
}
```

## Configuration

To configure our Mongo cluster we will use Mongo console client (https://www.mongodb.org/downloads#production please use at least version 3.0!).

1. Connect to master node (use credentials described above):

    ```
    $ mongo --host $node0Host --port $node0Port -u $username -p $password --authenticationDatabase $dbname
    ```
2. Now you are in Mongo console, put following calls:

    ```
    > rs.initiate()
    > conf=rs.conf()
    > conf.members[0].host="$node0Host:$node0Port"
    > rs.reconfig(conf)
    > rs.add("$node1Host:$node1Port")
    > rs.add("$node2Host:$node2Port")
    ```
4. You can verify you confiuration by check status in Mongo console - "statestr" value should be equal to "PRIMARY" or "SECONDARY":

    ```
    > rs.status()
    ```
3. Your replica is configured. You can now start using it by creating new Db schema and users.
4. REMEMBER! From now, during every connection to Mongo with your client you need to use URI with all 3 addresses:

    ```
    $ mongo -u $username -p $password --host $replicaSetName/$node0Host:$node0Port,$node1Host:$node1Port,$node2Host:$node2Port $dbname
    ```

## WARN!
Above instruction requires internal access to CF (e.g. by deployed apps).

Please expose service first and then use its external address to configure cluster from outside of CF.
