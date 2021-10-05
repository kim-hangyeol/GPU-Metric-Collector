link=stats/summary
#link=stats
#link=pods
#link=metrics

#APISERVER=$(kubectl config view --minify --context $cluster | grep server | cut -f 2- -d ":" | tr -d " ")
SECRET_NAME=$(kubectl get secrets | grep metric-test | cut -f1 -d ' ')
TOKEN=$(kubectl describe secret $SECRET_NAME | grep -E '^token' | cut -f2 -d':' | tr -d " ")

curl https://10.0.5.21:10250/$link --header "Authorization: Bearer $TOKEN" --insecure
