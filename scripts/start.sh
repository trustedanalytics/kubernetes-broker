export K8S_API_PORT=8080
export VCAP_APPLICATION='{"port":8081, "host":"0.0.0.0"}'

export AUTH_USER="admin"
export AUTH_PASS="password"

export INSECURE_SKIP_VERIFY=true
export ACCEPT_INCOMPLETE=false

export MAX_ORG_QUOTA=10
export BROKER_LOG_LEVEL=DEBUG

export VCAP_SERVICES='{
"user-provided": [{
    "credentials": {
     "apiEndpoint": "http://random_endpoint_.eu",
     "clientId": "clientRandomId",
     "clientSecret": "clientRandomSecret",
     "tokenUri": "http://random.eu/oauth/token",
     "tokenKey": "http://uaa.gotapaas.eu/token_key"
    },
    "label": "user-provided",
    "name": "sso"
},
{
    "credentials": {
     "password": "admin",
     "url": "kube-tap.dev-krb.gotapaas.eu",
     "username": "admin"
    },
    "label": "user-provided",
    "name": "kubernetes-creator-credentials"
   }]
}'

$GOPATH/bin/kubernetes-broker
exit $?
