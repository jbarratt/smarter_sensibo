## A smarter smart air conditioner

I got a Sensibo to control my mini-split. It *almost* does what I want.

I found a blog post of a person who had the same problem and came up with a workaround.


[sensibo scheduler](https://www.proxiblue.com.au/blog/ifttt-sensibo-climate-react-schedule/)

What I'd like is a little more sophisticated. It's often cold in the morning and warm in the afternoon; and I tend to get up early and exercise in the space.

* If it's the weekend, I'd like it to be off.
* If it's between 5am and 5pm on a weekday, I'd like my room to be climate controlled.
	* If it's under 70 degrees, I'd like it to warm to 70 and shut off
	* If it's over 70 degrees, I'd like it to cool to 74 and then shut off.



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

## TODO

Add a dead man switch metric and alarm so it's clear when the app fails to execute 3 times in a row

### API method

[API documentation](https://sensibo.github.io/)

There is an old python client available at https://github.com/Sensibo/sensibo-python-sdk/blob/master/sensibo_client.py but it doesn't have the magic climate settings.

You authenticate to the Sensibo API by providing your api key in the request query parameters as `?apiKey={your_api_key}`

```
	curl -X GET https://home.sensibo.com/api/v2/users/me/pods?fields=*&apiKey={api_key}
	RMrG4ctV
	result[0].acState.on = [true|false]
	# in celcius
	result[0].measurements.temperature

	Calling the pods? endpoint with fields=id,acState,measurements,smartMode gives what we need for basic current temp and if the unit is off or on.

	http get https://home.sensibo.com/api/v2/pods/$SENSIBO_DEVICE/smartmode?apiKey=$SENSIBO_API_KEY
```

Theoretically I should be able to do a PATCH to smartmode setting a few values.

Turns out it actually needs a POST but it can be incomplete data.

POST to /api/v2/pods/$device/smartmode?apiKey=..., see `patch_body.json` for examples.

So, a good workflow seems to be:

1. load the bits of state that we care about into known types
2. figure out what the state should be
3. Update the state
4. POST the new state
  * /pods/$device/acStates/
  * /pods/$device/smartmode/

Here's the JSON payload for the get on pods.
I dumped it into https://mholt.github.io/json-to-go/ as a starting point.

```
{
  "status": "success",
  "result": [
    {
      "acState": {
        "on": false,
        "fanLevel": "high",
        "temperatureUnit": "F",
        "targetTemperature": 75,
        "mode": "heat",
        "swing": "rangeFull"
      },
      "measurements": {
        "batteryVoltage": null,
        "temperature": 25.5,
        "humidity": 48.2,
        "time": {
          "secondsAgo": 25,
          "time": "2019-06-01T05:30:36.417685Z"
        },
        "rssi": "-51",
        "piezo": [
          null,
          null
        ]
      },
      "smartMode": {
        "deviceUid": "...",
        "highTemperatureWebhook": null,
        "highTemperatureThreshold": 23.3333333333333,
        "lowTemperatureWebhook": null,
        "type": "temperature",
        "lowTemperatureState": {
          "on": false,
          "fanLevel": "strong",
          "temperatureUnit": "F",
          "targetTemperature": 75,
          "mode": "heat"
        },
        "enabled": true,
        "highTemperatureState": {
          "on": false,
          "fanLevel": "strong",
          "temperatureUnit": "F",
          "targetTemperature": 71,
          "mode": "cool"
        },
        "lowTemperatureThreshold": 21.6666666666667
      },
      "id": "..."
    }
  ]
}
```
