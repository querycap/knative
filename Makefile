gen:
	go run main.go

apply-operator:
	kubectl apply -f ./operator/operator.yaml

apply-serving:
	kubectl apply -f ./serving.yaml

remove-serving:
	kubectl delete -f ./serving.yaml

