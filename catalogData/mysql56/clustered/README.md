# Clustered MySql

## Preparation

This service is served as MySql Percona XtraDB cluster. It uses fixed number of nodes (currently 3).

This is example of environments returned by binded app:
```json
{
  "VCAP_SERVICES": {
    "mysql56-multinode": [
      {
        "credentials": {
          "dbname": "17tWMJiPwa",
          "password": "PnwUYmYPt9",
          "username": "sUkwJiZoEh",
          "uri": "mysql:loadbalance://sUkwJiZoEh:PnwUYmYPt9@ip-10-10-3-201.ec2.internal:30274,ip-10-10-3-201.ec2.internal:30263,ip-10-10-3-201.ec2.internal:32045/17tWMJiPwa",
          "nodes": [
            {
              "host": "ip-10-10-3-201.ec2.internal",
              "ports": {
                "3306/tcp": "30274"
              },
              "nodeName": "xa33b437050ee4-node1"
            },
            {
              "host": "ip-10-10-3-201.ec2.internal",
              "ports": {
                "3306/tcp": "30263"
              },
              "nodeName": "xa33b437050ee4-node2"
            },
            {
              "host": "ip-10-10-3-201.ec2.internal",
              "ports": {
                "3306/tcp": "32045"
              },
              "nodeName": "xa33b437050ee4-node3"
            }
          ]
        },
        "label": "mysql56-multinode",
        "name": "test-service-mysql",
        "plan": "simple",
        "tags": [
           "mysql56",
           "mysql",
           "relational",
           "k8s",
           "clustered"
        ]
      }
    ]
  }
}
```
## Checking cluster configuration

In order to verify if cluster is up and configured you should:

1. Connect to one of cluster nodes via MySql client
2. Execute following command:
```
mysql> show status like 'wsrep_cluster_size';
```

Result should looks like:

```
+--------------------+-------+
| Variable_name      | Value |
+--------------------+-------+
| wsrep_cluster_size | 3     |
+--------------------+-------+
1 row in set (0.06 sec)
```

## Running

In order to establish connection with MySql cluster you should connect with one or more xa33b437050ee4-nodes.

Example jdbc url should looks like:

```
jdbc:mysql:loadbalance://ip-10-10-3-201.ec2.internal:32045,ip-10-10-3-201.ec2.internal:30263,ip-10-10-3-201.ec2.internal:30274/17tWMJiPwa
```
