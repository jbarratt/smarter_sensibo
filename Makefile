.PHONY: deps clean build
export GO111MODULE=on
BINARY_NAME=handler
BUCKET_NAME=serialized-sam-deploy

all: deps build
build:
	cd code && GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) cmd/smarter_sensibo/main.go

deps:
	cd code && go get -u ./...

clean: 
	rm -rf ./code/handler

package:
	aws-vault exec serialized -- sam package --output-template-file packaged.yaml --s3-bucket $(BUCKET_NAME)

deploy:
	aws-vault exec --no-session serialized -- sam deploy --template-file packaged.yaml --stack-name sensibo-app --capabilities CAPABILITY_IAM
