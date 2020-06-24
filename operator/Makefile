VERSION=$(shell cat .version)

sync:
	wget -O - https://github.com/knative/operator/releases/download/${VERSION}/operator.yaml | sed 's/image: gcr\.io\/knative[0-9a-z/.-]*/image: querycap\/knative-operator\:'${VERSION}' #/g' > operator.yaml
