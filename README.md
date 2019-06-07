## A smarter smart air conditioner

There's a blog post describing what's happening here:

[Making a Smart Thermostat Smarter](https://serialized.net/2019/06/smarter-smart/)


## Needed tools

### API key

* Get an API key via Sensibo online account: https://home.sensibo.com/me/api
* Get the wanted Device ID from your listed devices: https://home.sensibo.com/#/pods

## Store the API key

```
aws ssm put-parameter --name '/keys/sensibo' --value "your key" --type "SecureString"
```

## Build and deploy infrastructure

I'm using SAM here because it's a really simple way to get some code executing.

```
$ pip3 install --user aws-sam-cli
$ sam init --runtime go1.x
$ aws s3 mb (some deployment bucket)
```
Then, update the `Makefile` so it has your bucket.

Then it's ready to deploy:

```
$ make build
$ make package
$ make deploy
```

