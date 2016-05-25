# Clustered Postgresql

## Preparation

This service is served as three nodes Postgresql cluster. Cluster consist of one master node, two slaves nodes and one node with pgpool application, 
configured to served as a load balancer for three postgresql nodes.

Read/writes requests can be directly send to master node. Slaves nodes can only process read requests. Pgpool is configured to redirect writes request only to master node.


This is example of environments returned by binded app:
```json
{
  "VCAP_SERVICES": {
    "postgresql94-multinode": [
        {
          "label": "postgresql94-multinode",
          "name": "custom-postgresql",
          "plan": "free",
          "tags": [
            "postgresql94",
            "postgresql",
            "relational",
            "k8s",
            "clustered"
          ],
          "credentials": {
            "username": "z2Vc2vfBwI",
            "password": "6VDZo8y5l3",
            "dbname": "7NSr8z2HBR",
            "nodes": [
              {
                "nodeName": "x58f949c3baa34-master",
                "host": "x58f949c3baa34-master.service.consul",
                "ports": {
                  "5432/tcp": "30126"
                }
              },
              {
                "nodeName": "x58f949c3baa34-pgpool",
                "host": "x58f949c3baa34-pgpool.service.consul",
                "ports": {
                  "5432/tcp": "31450"
                }
              },
              {
                "nodeName": "x58f949c3baa34-slave-1",
                "host": "x58f949c3baa34-slave-1.service.consul",
                "ports": {
                  "5432/tcp": "31239"
                }
              },
              {
                "nodeName": "x58f949c3baa34-slave-2",
                "host": "x58f949c3baa34-slave-2.service.consul",
                "ports": {
                  "5432/tcp": "32538"
                }
              }
            ]
          }
        }
      ]
  }
}
```

## Running

The simples way to establish connection with postgresql cluster is to  connect with x58f949c3baa34-pgpool node
(then requests will be load balanced within 3 running nodes).

Example jdbc url should looks like:

```
jdbc:postgresql://xbd8fc7ece8614-pgpool.service.consul:31642/GbjkdiGMFA
```
