k port-forward sfh-db-mock-deployment 3000 # demoPortForward

curl -X 'POST' \
  'http://localhost:3000/provider' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "clientId": "edgizer-6sr7a",
  "clientSecret": "CLIENT_SECRET_6SR7A"
}' # providerPostReq

curl -X 'POST' \
  'http://localhost:3000/cluster' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "providerId": 1,
  "kubeconf": "kubeconf",
  "imagePullSecretNames": "secret",
  "registryPath": "thisispath",
  "daprAppId": "dapr"
}' # clusterPostReq

curl -X 'DELETE' \
  'http://localhost:3000/cluster/1' \
  -H 'accept: */*' # clusterDeleteReq