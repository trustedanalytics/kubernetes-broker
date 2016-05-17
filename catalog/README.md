# Dynamic Offering

New element in catalog can be added dynamicly with HTTP request and does not required any broker restarts:

1) Add new service using PUT call on /v2/dynamicservice - exampled request is placed below
2) Use CLI cf client to enable access to new created service:
```
cf enable-service-access dynamic-mongo
```
3) New service is available in marketplace now. New instances can be spawn.

## Exampled request

```json
{
 "organization_guid": "fcb5307e-4995-4eeb-bfcd-08b3d3023bb5",
 "space_guid": "88d4676a-e374-4e36-9814-c269b85c5a0f",
 "parameters": null,
 "updateBroker": true,
 "dynamicService": {
   "serviceName": "dynamic-mongo",
   "planName": "dynamic-mongo",
   "isPlanFree": true,
   "containers": [
     {
       "name": "k-mongodb30",
       "image": "frodenas/mongodb:3.0",
       "ports": [
         {
           "containerPort": 27017,
           "protocol": "TCP"
         }
       ],
       "env": [
         { "name": "MANAGED_BY", "value":"TAP" },
         { "name": "MONGODB_PASSWORD",   "value": "user" },
         { "name": "MONGODB_USERNAME",   "value": "password" },
         { "name": "MONGODB_DBNAME",   "value": "test" }
       ],
       "resources": {},
       "imagePullPolicy": ""
     }
   ],
   "servicePorts": [
     {
       "name": "",
       "protocol": "TCP",
       "port": 27017,
       "targetPort": 0,
       "nodePort": 0
     }
   ],
   "credentialMappings": {
     "dbname": "$env_MONGODB_DBNAME",
     "hostname": "$hostname",
     "password": "$env_MONGODB_PASSWORD",
     "port": "$port_27017",
     "ports": {
       "27017/tcp": "$port_27017"
     },
     "uri": "mongodb://$env_MONGODB_USERNAME:$env_MONGODB_PASSWORD@$hostname:$port_27017/$env_MONGODB_DBNAME",
     "username": "$env_MONGODB_USERNAME"
   }
 }
}
```

Fields:
* 'updateBroker' - will enforce cf borker-update
* 'containers' - list of kubernetes containers - [definition](http://kubernetes.io/docs/api-reference/v1/definitions/#_v1_container)
* 'servicePorts' - list of ports which will be assigned to kubernetes service - [definition](http://kubernetes.io/docs/api-reference/v1/definitions/#_v1_serviceport).
This is a simple mapping, where 'port' value refers to 'containerPort' value
* 'credentialMappings' - map of envs which will be serve to other apps. To put there container env value, following pattern has to be fulfill: 'env_ENV_NAME'.
Service NodePort value also can be use. To access it use pattern: 'port_PORT_NUMBER'